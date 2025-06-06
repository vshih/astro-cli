package settings

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	astrocore "github.com/astronomer/astro-cli/astro-client-core"
	"github.com/astronomer/astro-cli/docker"
	"github.com/astronomer/astro-cli/pkg/fileutil"
	"github.com/astronomer/astro-cli/pkg/logger"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

var (
	// ConfigFileName is the name of the config files (home / project)
	// ConfigFileName = "airflow_settings"
	// ConfigFileType is the config file extension
	ConfigFileType = "yaml"
	// WorkingPath is the path to the working directory
	WorkingPath, _ = fileutil.GetWorkingDir()

	// viperSettings is the viper object in a project directory
	viperSettings *viper.Viper

	settings Config

	// AirflowVersionTwo 2.0.0
	AirflowVersionTwo uint64 = 2

	// Monkey patched as of now to write unit tests
	// TODO: do replace this with interface based mocking once changes are in place in `airflow` package
	execAirflowCommand = docker.AirflowCommand
)

const (
	airflowConnectionList = "airflow connections list -o yaml"
	airflowPoolsList      = "airflow pools list -o yaml"
	airflowConnExport     = "airflow connections export tmp.connections --file-format env"
	airflowVarExport      = "airflow variables export tmp.var"
	catVarFile            = "cat tmp.var"
	rmVarFile             = "rm tmp.var"
	catConnFile           = "cat tmp.connections"
	configReadErrorMsg    = "Error reading Airflow Settings file. Connections, Variables, and Pools were not loaded please check your Settings file syntax: %s\n"
	noColorString         = "[\u001B\u009B][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[a-zA-Z\\d]*)*)?\u0007)|(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PRZcf-ntqry=><~]))"
)

var (
	errNoID = errors.New("container ID is not found, the webserver may not be running")
	re      = regexp.MustCompile(noColorString)
)

// ConfigSettings is the main builder of the settings package
func ConfigSettings(id, settingsFile string, envConns map[string]astrocore.EnvironmentObjectConnection, version uint64, connections, variables, pools bool) error {
	if id == "" {
		return errNoID
	}
	err := InitSettings(settingsFile)
	if err != nil {
		return err
	}
	if pools {
		if err := AddPools(id, version); err != nil {
			return fmt.Errorf("error adding pools: %w", err)
		}
	}
	if variables {
		if err := AddVariables(id, version); err != nil {
			return fmt.Errorf("error adding variables: %w", err)
		}
	}
	if connections {
		if err := AddConnections(id, version, envConns); err != nil {
			return fmt.Errorf("error adding connections: %w", err)
		}
	}
	return nil
}

// InitSettings initializes settings file
func InitSettings(settingsFile string) error {
	// Set up viper object for project config
	viperSettings = viper.New()
	ConfigFileName := strings.Split(settingsFile, ".")[0]
	viperSettings.SetConfigName(ConfigFileName)
	viperSettings.SetConfigType(ConfigFileType)
	workingConfigFile := filepath.Join(WorkingPath, fmt.Sprintf("%s.%s", ConfigFileName, ConfigFileType))
	// Add the path we discovered
	viperSettings.SetConfigFile(workingConfigFile)

	// Read in project config
	readErr := viperSettings.ReadInConfig()

	if readErr != nil {
		fmt.Printf(configReadErrorMsg, readErr)
	}

	err := viperSettings.Unmarshal(&settings)
	// Try and use old settings file if error
	if err != nil {
		return errors.Wrap(err, "unable to decode file")
	}
	return nil
}

// AddVariables is a function to add Variables from settings.yaml
func AddVariables(id string, version uint64) error {
	variables := settings.Airflow.Variables
	for _, variable := range variables {
		if !objectValidator(0, variable.VariableName) {
			if objectValidator(0, variable.VariableValue) {
				fmt.Print("Skipping Variable Creation: No Variable Name Specified.\n")
			}
		} else if objectValidator(0, variable.VariableValue) {
			baseCmd := "airflow variables "
			if version >= AirflowVersionTwo {
				baseCmd += "set %s " // Airflow 2.0.0 command
			} else {
				baseCmd += "-s %s"
			}

			airflowCommand := fmt.Sprintf(baseCmd, variable.VariableName)

			airflowCommand += fmt.Sprintf("'%s'", variable.VariableValue)
			out, err := execAirflowCommand(id, airflowCommand)
			if err != nil {
				return fmt.Errorf("error adding variable %s: %w", variable.VariableName, err)
			}
			logger.Debugf("Adding variable logs:\n%s", out)
			fmt.Printf("Added Variable: %s\n", variable.VariableName)
		}
	}
	return nil
}

// AddConnections is a function to add Connections from settings.yaml
func AddConnections(id string, version uint64, envConns map[string]astrocore.EnvironmentObjectConnection) error {
	connections := settings.Airflow.Connections
	connections = AppendEnvironmentConnections(connections, envConns)

	baseCmd := "airflow connections "
	var baseRmCmd, baseListCmd, connIDArg string
	if version >= AirflowVersionTwo {
		// Airflow 2.0.0 command
		// based on https://airflow.apache.org/docs/apache-airflow/2.0.0/cli-and-env-variables-ref.html
		baseRmCmd = baseCmd + "delete "
		baseListCmd = baseCmd + "list -o plain"
		connIDArg = ""
	} else {
		// Airflow 1.0.0 command based on
		// https://airflow.readthedocs.io/en/1.10.12/cli-ref.html#connections
		baseRmCmd = baseCmd + "-d "
		baseListCmd = baseCmd + "-l "
		connIDArg = "--conn_id"
	}
	airflowCommand := baseListCmd
	out, err := execAirflowCommand(id, airflowCommand)
	if err != nil {
		return fmt.Errorf("error listing connections: %w", err)
	}

	for i := range connections {
		conn := connections[i]
		if !objectValidator(0, conn.ConnID) {
			continue
		}

		extraString := jsonString(&conn)

		quotedConnID := "'" + conn.ConnID + "'"

		if strings.Contains(out, quotedConnID) || strings.Contains(out, conn.ConnID) {
			fmt.Printf("Updating Connection %q...\n", conn.ConnID)
			airflowCommand = fmt.Sprintf("%s %s %q", baseRmCmd, connIDArg, conn.ConnID)
			_, err = execAirflowCommand(id, airflowCommand)
			if err != nil {
				return fmt.Errorf("error removing connection %s: %w", conn.ConnID, err)
			}
		}

		if !objectValidator(1, conn.ConnType, conn.ConnURI) {
			fmt.Printf("Skipping %s: conn_type or conn_uri must be specified.\n", conn.ConnID)
			continue
		}

		airflowCommand = prepareAirflowConnectionAddCommand(version, &conn, extraString)
		if airflowCommand != "" {
			out, err := execAirflowCommand(id, airflowCommand)
			if err != nil {
				return fmt.Errorf("error adding connection %s: %w", conn.ConnID, err)
			}
			logger.Debugf("Adding Connection logs:\n\n%s", out)
			fmt.Printf("Added Connection: %s\n", conn.ConnID)
		}
	}
	return nil
}

func prepareAirflowConnectionAddCommand(version uint64, conn *Connection, extraString string) string {
	if conn == nil {
		return ""
	}
	baseCmd := "airflow connections "
	var baseAddCmd, connIDArg, connTypeArg, connURIArg, connExtraArg, connHostArg, connLoginArg, connPasswordArg, connSchemaArg, connPortArg string
	if version >= AirflowVersionTwo {
		// Airflow 2.0.0 command
		// based on https://airflow.apache.org/docs/apache-airflow/2.0.0/cli-and-env-variables-ref.html
		baseAddCmd = baseCmd + "add "
		connIDArg = ""
		connTypeArg = "--conn-type"
		connURIArg = "--conn-uri"
		connExtraArg = "--conn-extra"
		connHostArg = "--conn-host"
		connLoginArg = "--conn-login"
		connPasswordArg = "--conn-password"
		connSchemaArg = "--conn-schema"
		connPortArg = "--conn-port"
	} else {
		// Airflow 1.0.0 command based on
		// https://airflow.readthedocs.io/en/1.10.12/cli-ref.html#connections
		baseAddCmd = baseCmd + "-a "
		connIDArg = "--conn_id"
		connTypeArg = "--conn_type"
		connURIArg = "--conn_uri"
		connExtraArg = "--conn_extra"
		connHostArg = "--conn_host"
		connLoginArg = "--conn_login"
		connPasswordArg = "--conn_password"
		connSchemaArg = "--conn_schema"
		connPortArg = "--conn_port"
	}
	var j int
	airflowCommand := fmt.Sprintf("%s %s '%s' ", baseAddCmd, connIDArg, conn.ConnID)
	if objectValidator(0, conn.ConnType) {
		airflowCommand += fmt.Sprintf("%s '%s' ", connTypeArg, conn.ConnType)
		j++
	}
	if extraString != "" {
		airflowCommand += fmt.Sprintf("%s '%s' ", connExtraArg, extraString)
	}
	if objectValidator(0, conn.ConnHost) {
		airflowCommand += fmt.Sprintf("%s '%s' ", connHostArg, conn.ConnHost)
		j++
	}
	if objectValidator(0, conn.ConnLogin) {
		airflowCommand += fmt.Sprintf("%s '%s' ", connLoginArg, conn.ConnLogin)
		j++
	}
	if objectValidator(0, conn.ConnPassword) {
		airflowCommand += fmt.Sprintf("%s '%s' ", connPasswordArg, conn.ConnPassword)
		j++
	}
	if objectValidator(0, conn.ConnSchema) {
		airflowCommand += fmt.Sprintf("%s '%s' ", connSchemaArg, conn.ConnSchema)
		j++
	}
	if conn.ConnPort != 0 {
		airflowCommand += fmt.Sprintf("%s %v", connPortArg, conn.ConnPort)
		j++
	}
	if objectValidator(0, conn.ConnURI) && j == 0 {
		airflowCommand += fmt.Sprintf("%s '%s' ", connURIArg, conn.ConnURI)
	}
	return airflowCommand
}

func AppendEnvironmentConnections(connections Connections, envConnections map[string]astrocore.EnvironmentObjectConnection) Connections {
	for envConnID, envConn := range envConnections {
		for i := range connections {
			if connections[i].ConnID == envConnID {
				// if connection already exists in settings file, skip it because the file takes precedence
				continue
			}
		}
		conn := Connection{
			ConnID:   envConnID,
			ConnType: envConn.Type,
		}
		if envConn.Host != nil {
			conn.ConnHost = *envConn.Host
		}
		if envConn.Port != nil {
			conn.ConnPort = *envConn.Port
		}
		if envConn.Login != nil {
			conn.ConnLogin = *envConn.Login
		}
		if envConn.Password != nil {
			conn.ConnPassword = *envConn.Password
		}
		if envConn.Schema != nil {
			conn.ConnSchema = *envConn.Schema
		}
		if envConn.Extra != nil {
			extra := make(map[any]any)
			for k, v := range *envConn.Extra {
				extra[k] = v
			}
			conn.ConnExtra = extra
		}
		connections = append(connections, conn)
	}
	return connections
}

// AddPools  is a function to add Pools from settings.yaml
func AddPools(id string, version uint64) error {
	pools := settings.Airflow.Pools
	baseCmd := "airflow "

	if version >= AirflowVersionTwo {
		// Airflow 2.0.0 command
		// based on https://airflow.apache.org/docs/apache-airflow/2.0.0/cli-and-env-variables-ref.html
		baseCmd += "pools set "
	} else {
		baseCmd += "pool -s "
	}

	for _, pool := range pools {
		if objectValidator(0, pool.PoolName) {
			airflowCommand := fmt.Sprintf("%s %s ", baseCmd, pool.PoolName)
			if pool.PoolSlot != 0 {
				airflowCommand += fmt.Sprintf("%v ", pool.PoolSlot)
				if objectValidator(0, pool.PoolDescription) {
					airflowCommand += fmt.Sprintf("'%s' ", pool.PoolDescription)
				} else {
					airflowCommand += "''"
				}
				fmt.Println(airflowCommand)
				out, err := execAirflowCommand(id, airflowCommand)
				if err != nil {
					return fmt.Errorf("error adding pool %s: %w", pool.PoolName, err)
				}
				logger.Debugf("Adding pool logs:\n%s", out)
				fmt.Printf("Added Pool: %s\n", pool.PoolName)
			} else {
				fmt.Printf("Skipping %s: Pool Slot must be set.\n", pool.PoolName)
			}
		}
	}
	return nil
}

func objectValidator(bound int, args ...string) bool {
	count := 0
	for _, arg := range args {
		if arg == "" {
			count++
		}
	}
	return count <= bound
}

func EnvExport(id, envFile string, version uint64, connections, variables bool) error {
	if id == "" {
		return errNoID
	}
	var parseErr bool
	if version >= AirflowVersionTwo {
		// env export variables if variables is true
		if variables {
			err := EnvExportVariables(id, envFile)
			if err != nil {
				fmt.Println(err)
				parseErr = true
			}
		}
		// env export connections if connections is true
		if connections {
			err := EnvExportConnections(id, envFile)
			if err != nil {
				fmt.Println(err)
				parseErr = true
			}
		}
		if parseErr {
			return errors.New("there was an error during env export")
		}
		return nil
	}

	return errors.New("Command must be used with Airflow 2.X")
}

func EnvExportVariables(id, envFile string) error {
	// setup airflow command to export variables
	out, err := execAirflowCommand(id, airflowVarExport)
	if err != nil {
		return fmt.Errorf("error exporting variables: %w", err)
	}
	logger.Debugf("Env Export Variables logs:\n\n%s", out)

	if strings.Contains(out, "successfully") {
		// get variables from file created by airflow command
		out, err = execAirflowCommand(id, catVarFile)
		if err != nil {
			return fmt.Errorf("error reading variables file: %w", err)
		}

		m := map[string]string{}
		err := json.Unmarshal([]byte(out), &m)
		if err != nil {
			fmt.Printf("variable json decode unsuccessful: %s", err.Error())
		}
		// add variables to the env file
		f, err := os.OpenFile(envFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644) //nolint:mnd
		if err != nil {
			return errors.Wrap(err, "Writing variables to file unsuccessful")
		}

		defer f.Close()

		for k, v := range m {
			fmt.Println("Exporting Variable: " + k)
			_, err := f.WriteString("\nAIRFLOW_VAR_" + strings.ToUpper(k) + "=" + v)
			if err != nil {
				fmt.Printf("error adding variable %s to file: %s\n", k, err.Error())
			}
		}
		fmt.Println("Aiflow variables successfully export to the file " + envFile + "\n")
		_, err = execAirflowCommand(id, rmVarFile)
		if err != nil {
			return fmt.Errorf("error removing variables file: %w", err)
		}
		return nil
	}
	return errors.New("variable export unsuccessful")
}

func EnvExportConnections(id, envFile string) error {
	// Airflow command to export connections to env uris
	out, err := execAirflowCommand(id, airflowConnExport)
	if err != nil {
		return fmt.Errorf("error exporting connections: %w", err)
	}
	logger.Debugf("Env Export Connections logs:\n%s", out)

	if strings.Contains(out, "successfully") {
		// get connections from file craeted by airflow command
		out, err = execAirflowCommand(id, catConnFile)
		if err != nil {
			return fmt.Errorf("error reading connections file: %w", err)
		}

		vars := strings.Split(out, "\n")
		// add connections to the env file
		f, err := os.OpenFile(envFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644) //nolint:mnd
		if err != nil {
			return errors.Wrap(err, "Writing connections to file unsuccessful")
		}

		defer f.Close()

		for i := range vars {
			varSplit := strings.SplitN(vars[i], "=", 2) //nolint:mnd
			if len(varSplit) > 1 {
				fmt.Println("Exporting Connection: " + varSplit[0])
				_, err := f.WriteString("\nAIRFLOW_CONN_" + strings.ToUpper(varSplit[0]) + "=" + varSplit[1])
				if err != nil {
					fmt.Printf("error adding connection %s to file: %s\n", varSplit[0], err.Error())
				}
			}
		}
		fmt.Println("Aiflow connections successfully export to the file " + envFile + "\n")
		rmCmd := "rm tmp.connection"
		_, err = execAirflowCommand(id, rmCmd)
		if err != nil {
			return fmt.Errorf("error removing connections file: %w", err)
		}
		return nil
	}
	return errors.New("connection export unsuccessful")
}

func Export(id, settingsFile string, version uint64, connections, variables, pools bool) error {
	if id == "" {
		return errNoID
	}
	// init settings file
	err := InitSettings(settingsFile)
	if err != nil {
		return err
	}
	var parseErr bool
	// export Airflow Objects
	if version < AirflowVersionTwo {
		return errors.New("Command must be used with Airflow 2.X")
	}
	if pools {
		err = ExportPools(id)
		if err != nil {
			fmt.Println(err)
			parseErr = true
		}
	}
	if variables {
		err = ExportVariables(id)
		if err != nil {
			fmt.Println(err)
			parseErr = true
		}
	}
	if connections {
		err := ExportConnections(id)
		if err != nil {
			fmt.Println(err)
			parseErr = true
		}
	}
	if parseErr {
		return errors.New("there was an error during export")
	}
	return nil
}

func ExportConnections(id string) error {
	// Setup airflow command to export connections
	out, err := execAirflowCommand(id, airflowConnectionList)
	if err != nil {
		return fmt.Errorf("error listing connections: %w", err)
	}
	logger.Debugf("Export Connections logs:\n%s", out)
	// remove all color from output of the airflow command
	plainOut := re.ReplaceAllString(out, "")
	// remove extra warning text
	yamlCons := "- conn_id:" + strings.SplitN(plainOut, "- conn_id:", 2)[1] //nolint:mnd

	var connections AirflowConnections

	err = yaml.Unmarshal([]byte(yamlCons), &connections)
	if err != nil {
		return err
	}
	// add connections to settings file
	for i := range connections {
		var port int
		if connections[i].ConnPort != "" {
			port, err = strconv.Atoi(connections[i].ConnPort)
			if err != nil {
				fmt.Printf("Issue with parsing port number: %s", err.Error())
			}
		}
		for j := range settings.Airflow.Connections {
			if settings.Airflow.Connections[j].ConnID == connections[i].ConnID {
				fmt.Println("Updating Connection: " + connections[i].ConnID)
				// Remove connection if it already exits
				settings.Airflow.Connections = append(settings.Airflow.Connections[:j], settings.Airflow.Connections[j+1:]...)
				break
			}
		}
		fmt.Println("Exporting Connection: " + connections[i].ConnID)

		newConnection := Connection{
			ConnID:       connections[i].ConnID,
			ConnType:     connections[i].ConnType,
			ConnHost:     connections[i].ConnHost,
			ConnSchema:   connections[i].ConnSchema,
			ConnLogin:    connections[i].ConnLogin,
			ConnPassword: connections[i].ConnPassword,
			ConnPort:     port,
			ConnExtra:    connections[i].ConnExtra,
		}

		settings.Airflow.Connections = append(settings.Airflow.Connections, newConnection)
	}
	// write to settings file
	viperSettings.Set("airflow", settings.Airflow)
	err = viperSettings.WriteConfig()
	if err != nil {
		return err
	}
	fmt.Printf("successfully exported Connections\n\n")
	return nil
}

func ExportVariables(id string) error {
	// setup files
	out, err := execAirflowCommand(id, airflowVarExport)
	if err != nil {
		return fmt.Errorf("error exporting variables: %w", err)
	}
	logger.Debugf("Export Variables logs:\n%s", out)

	if strings.Contains(out, "successfully") {
		// get variables created by the airflow command
		out, err = execAirflowCommand(id, catVarFile)
		if err != nil {
			return fmt.Errorf("error reading variables file: %w", err)
		}

		var m map[string]interface{}
		err := json.Unmarshal([]byte(out), &m)
		if err != nil {
			fmt.Println("variable json decode unsuccessful")
		}
		// add the variables to settings object
		for k, v := range m {
			for j := range settings.Airflow.Variables {
				if settings.Airflow.Variables[j].VariableName == k {
					fmt.Println("Updating Pool: " + k)
					// Remove variable if it already exists
					settings.Airflow.Variables = append(settings.Airflow.Variables[:j], settings.Airflow.Variables[j+1:]...)
					break
				}
			}

			var vs string
			switch vt := v.(type) {
			case string:
				vs = vt
			default:
				// Re-encode complex types as JSON.
				b, err := json.Marshal(v)
				if err != nil {
					fmt.Println("variable json reencode unsuccessful")
					return err
				}
				vs = string(b)
			}
			newVariables := Variables{{k, vs}}
			fmt.Println("Exporting Variable: " + k)
			settings.Airflow.Variables = append(settings.Airflow.Variables, newVariables...)
		}
		// write variables to settings file
		viperSettings.Set("airflow", settings.Airflow)
		err = viperSettings.WriteConfig()
		if err != nil {
			return err
		}
		_, err = execAirflowCommand(id, rmVarFile)
		if err != nil {
			return fmt.Errorf("error removing variables file: %w", err)
		}
		fmt.Printf("successfully exported variables\n\n")
		return nil
	}
	return errors.New("variable export unsuccessful")
}

func ExportPools(id string) error {
	// Setup airflow command to export pools
	airflowCommand := airflowPoolsList
	out, err := execAirflowCommand(id, airflowCommand)
	if err != nil {
		return fmt.Errorf("error listing pools: %w", err)
	}
	logger.Debugf("Export Pools logs:\n%s", out)

	// remove all color from output of the airflow command
	plainOut := re.ReplaceAllString(out, "")

	var pools AirflowPools
	// remove warnings and extra text from the the output
	yamlpools := "- description:" + strings.SplitN(plainOut, "- description:", 2)[1] //nolint:mnd

	err = yaml.Unmarshal([]byte(yamlpools), &pools)
	if err != nil {
		return err
	}
	// add pools to the settings object
	for i := range pools {
		if pools[i].PoolName != "default_pool" {
			continue
		}
		slot, err := strconv.Atoi(pools[i].PoolSlot)
		if err != nil {
			fmt.Println("Issue with parsing pool slot number: ")
			fmt.Println(err)
		}
		for j := range settings.Airflow.Pools {
			if settings.Airflow.Pools[j].PoolName == pools[i].PoolName {
				fmt.Println("Updating Pool: " + pools[i].PoolName)
				// Remove pool if it already exits
				settings.Airflow.Pools = append(settings.Airflow.Pools[:j], settings.Airflow.Pools[j+1:]...)
				break
			}
		}
		fmt.Println("Exporting Pool: " + pools[i].PoolName)
		newPools := Pools{{pools[i].PoolName, slot, pools[i].PoolDescription}}
		settings.Airflow.Pools = append(settings.Airflow.Pools, newPools...)
	}
	// write pools to the airflow settings file
	viperSettings.Set("airflow", settings.Airflow)
	err = viperSettings.WriteConfig()
	if err != nil {
		return err
	}
	fmt.Printf("successfully exported pools\n\n")
	return nil
}

func jsonString(conn *Connection) string {
	var extraMap map[string]any

	switch connExtra := conn.ConnExtra.(type) {
	case string:
		// if extra is already a string we assume it is a JSON-encoded extra string
		return connExtra
	case map[any]any:
		// the extra map is loaded as a map[any]any, but it needs to be map[string]any to be
		// marshaled to JSON, and for it to be a valid Airflow connection extra, so we convert it
		extraMap = make(map[string]any)
		for k, v := range connExtra {
			kStr, ok := k.(string)
			if !ok {
				fmt.Printf("Error asserting extra key as string for %s, found type: %T\n", conn.ConnID, k)
				continue
			}
			extraMap[kStr] = v
		}
	case map[string]any:
		// if some future code provides a map[string]any, we can use that directly
		extraMap = connExtra
	case nil:
		// if the extra is nil, we proceed with an empty extra
		return ""
	default:
		// if the extra type is something else entirely, we log a warning and proceed with an empty extra
		fmt.Printf("Error converting extra to map for %s, found type: %T\n", conn.ConnID, conn.ConnExtra)
		return ""
	}

	// marshal the extra map to a JSON string
	extraBytes, err := json.Marshal(extraMap)
	if err != nil {
		fmt.Printf("Error marshaling extra for %s: %s\n", conn.ConnID, err.Error())
		return ""
	}
	return string(extraBytes)
}

func WriteAirflowSettingstoYAML(settingsFile string) error {
	err := InitSettings(settingsFile)
	if err != nil {
		return err
	}

	// Connections from settings file to connection YAML file
	connYAMLs := ConnYAMLs{}
	connections := settings.Airflow.Connections
	for i := range connections {
		newConnYAML := ConnYAML{
			ConnID:   connections[i].ConnID,
			ConnType: connections[i].ConnType,
			Host:     connections[i].ConnHost,
			Schema:   connections[i].ConnSchema,
			Login:    connections[i].ConnLogin,
			Password: connections[i].ConnPassword,
			Port:     connections[i].ConnPort,
			Extra:    connections[i].ConnExtra,
		}

		connYAMLs = append(connYAMLs, newConnYAML)
	}

	connectionsYAML := DAGRunConnections{
		ConnYAMLs: connYAMLs,
	}

	out, err := yaml.Marshal(connectionsYAML)
	if err != nil {
		fmt.Printf("Error creating connections from settings file: %s\n", err.Error())
	}

	err = fileutil.WriteStringToFile("./connections.yaml", string(out))
	if err != nil {
		fmt.Printf("Error creating connections from settings file:: %s\n", err.Error())
	}

	// Variables from settings file to variables YAML file
	varYAMLs := VarYAMLs{}
	variables := settings.Airflow.Variables
	for _, variable := range variables {
		newVarYAML := VarYAML{
			Key:   variable.VariableName,
			Value: variable.VariableValue,
		}

		varYAMLs = append(varYAMLs, newVarYAML)
	}

	variablesYAML := DAGRunVariables{
		VarYAMLs: varYAMLs,
	}

	out, err = yaml.Marshal(variablesYAML)
	if err != nil {
		fmt.Printf("Error creating variabels from settings file: %s\n", err.Error())
	}

	err = fileutil.WriteStringToFile("./variables.yaml", string(out))
	if err != nil {
		fmt.Printf("Error creating connections from settings file:: %s\n", err.Error())
	}

	return nil
}
