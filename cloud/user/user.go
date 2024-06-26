package user

import (
	httpContext "context"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	astrocore "github.com/astronomer/astro-cli/astro-client-core"
	"github.com/astronomer/astro-cli/config"
	"github.com/astronomer/astro-cli/context"
	"github.com/astronomer/astro-cli/pkg/input"
	"github.com/astronomer/astro-cli/pkg/printutil"

	"github.com/pkg/errors"
)

var (
	ErrInvalidRole             = errors.New("requested role is invalid. Possible values are ORGANIZATION_MEMBER, ORGANIZATION_BILLING_ADMIN and ORGANIZATION_OWNER ")
	ErrInvalidWorkspaceRole    = errors.New("requested role is invalid. Possible values are WORKSPACE_MEMBER, WORKSPACE_AUTHOR, WORKSPACE_OPERATOR and WORKSPACE_OWNER ")
	ErrInvalidOrganizationRole = errors.New("requested role is invalid. Possible values are ORGANIZATION_MEMBER, ORGANIZATION_BILLING_ADMIN and ORGANIZATION_OWNER ")
	ErrInvalidEmail            = errors.New("no email provided for the invite. Retry with a valid email address")
	ErrInvalidUserKey          = errors.New("invalid User selected")
	userPagnationLimit         = 100
	ErrUserNotFound            = errors.New("no user was found for the email you provided")
)

func CreateInvite(email, role string, out io.Writer, client astrocore.CoreClient) error {
	var (
		userInviteInput astrocore.CreateUserInviteRequest
		err             error
		ctx             config.Context
	)
	if email == "" {
		return ErrInvalidEmail
	}
	err = IsRoleValid(role)
	if err != nil {
		return err
	}
	ctx, err = context.GetCurrentContext()
	if err != nil {
		return err
	}
	userInviteInput = astrocore.CreateUserInviteRequest{
		InviteeEmail: email,
		Role:         role,
	}
	resp, err := client.CreateUserInviteWithResponse(httpContext.Background(), ctx.Organization, userInviteInput)
	if err != nil {
		return err
	}
	err = astrocore.NormalizeAPIError(resp.HTTPResponse, resp.Body)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "invite for %s with role %s created\n", email, role)
	return nil
}

func UpdateUserRole(email, role string, out io.Writer, client astrocore.CoreClient) error {
	var userID string
	err := IsRoleValid(role)
	if err != nil {
		return err
	}
	ctx, err := context.GetCurrentContext()
	if err != nil {
		return err
	}
	// Get all org users
	users, err := GetOrgUsers(client)
	if err != nil {
		return err
	}
	if email != "" {
		if err != nil {
			return err
		}

		for i := range users {
			if users[i].Username == email {
				userID = users[i].Id
			}
		}
		if userID == "" {
			return ErrUserNotFound
		}
	} else {
		user, err := SelectUser(users, "organization")
		userID = user.Id
		email = user.Username
		if err != nil {
			return err
		}
	}
	mutateUserInput := astrocore.MutateOrgUserRoleRequest{
		Role: role,
	}
	resp, err := client.MutateOrgUserRoleWithResponse(httpContext.Background(), ctx.Organization, userID, mutateUserInput)
	if err != nil {
		return err
	}
	err = astrocore.NormalizeAPIError(resp.HTTPResponse, resp.Body)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "The user %s role was successfully updated to %s\n", email, role)
	return nil
}

// IsRoleValid checks if the requested role is valid
// If the role is valid, it returns nil
// error errInvalidRole is returned if the role is not valid
func IsRoleValid(role string) error {
	validRoles := []string{"ORGANIZATION_MEMBER", "ORGANIZATION_BILLING_ADMIN", "ORGANIZATION_OWNER"}
	for _, validRole := range validRoles {
		if role == validRole {
			return nil
		}
	}
	return ErrInvalidRole
}

func SelectUser(users []astrocore.User, roleEntity string) (astrocore.User, error) {
	roleColumn := "ORGANIZATION ROLE"
	switch r := roleEntity; r {
	case "workspace":
		roleColumn = "WORKSPACE ROLE"
	case "deployment":
		roleColumn = "DEPLOYMENT ROLE"
	}

	table := printutil.Table{
		Padding:        []int{30, 50, 10, 50, 10, 10, 10},
		DynamicPadding: true,
		Header:         []string{"#", "FULLNAME", "EMAIL", "ID", roleColumn, "CREATE DATE"},
	}

	fmt.Println("\nPlease select the user:")

	userMap := map[string]astrocore.User{}
	for i := range users {
		index := i + 1
		switch r := roleEntity; r {
		case "deployment":
			table.AddRow([]string{
				strconv.Itoa(index),
				users[i].FullName,
				users[i].Username,
				users[i].Id,
				*users[i].DeploymentRole,
				users[i].CreatedAt.Format(time.RFC3339),
			}, false)
		case "workspace":
			table.AddRow([]string{
				strconv.Itoa(index),
				users[i].FullName,
				users[i].Username,
				users[i].Id,
				*users[i].WorkspaceRole,
				users[i].CreatedAt.Format(time.RFC3339),
			}, false)
		default:
			table.AddRow([]string{
				strconv.Itoa(index),
				users[i].FullName,
				users[i].Username,
				users[i].Id,
				*users[i].OrgRole,
				users[i].CreatedAt.Format(time.RFC3339),
			}, false)
		}

		userMap[strconv.Itoa(index)] = users[i]
	}

	table.Print(os.Stdout)
	choice := input.Text("\n> ")
	selected, ok := userMap[choice]
	if !ok {
		return astrocore.User{}, ErrInvalidUserKey
	}
	return selected, nil
}

// Returns a list of all of an organizations users
func GetOrgUsers(client astrocore.CoreClient) ([]astrocore.User, error) {
	offset := 0
	var users []astrocore.User

	ctx, err := context.GetCurrentContext()
	if err != nil {
		return nil, err
	}

	for {
		resp, err := client.ListOrgUsersWithResponse(httpContext.Background(), ctx.Organization, &astrocore.ListOrgUsersParams{
			Offset: &offset,
			Limit:  &userPagnationLimit,
		})
		if err != nil {
			return nil, err
		}
		err = astrocore.NormalizeAPIError(resp.HTTPResponse, resp.Body)
		if err != nil {
			return nil, err
		}
		users = append(users, resp.JSON200.Users...)

		if resp.JSON200.TotalCount <= offset {
			break
		}

		offset += userPagnationLimit
	}

	return users, nil
}

// Prints a list of all of an organizations users
func ListOrgUsers(out io.Writer, client astrocore.CoreClient) error {
	table := printutil.Table{
		Padding:        []int{30, 50, 10, 50, 10, 10, 10},
		DynamicPadding: true,
		Header:         []string{"FULLNAME", "EMAIL", "ID", "ORGANIZATION ROLE", "IDP MANAGED", "CREATE DATE"},
	}
	users, err := GetOrgUsers(client)
	if err != nil {
		return err
	}

	for i := range users {
		orgUserRelationIsIdpManaged := ""
		orgUserRelationIsIdpManagedPointer := users[i].OrgUserRelationIsIdpManaged
		if orgUserRelationIsIdpManagedPointer != nil {
			orgUserRelationIsIdpManaged = strconv.FormatBool(*users[i].OrgUserRelationIsIdpManaged)
		}
		table.AddRow([]string{
			users[i].FullName,
			users[i].Username,
			users[i].Id,
			*users[i].OrgRole,
			orgUserRelationIsIdpManaged,
			users[i].CreatedAt.Format(time.RFC3339),
		}, false)
	}

	table.Print(out)
	return nil
}

func AddWorkspaceUser(email, role, workspaceID string, out io.Writer, client astrocore.CoreClient) error {
	err := IsWorkspaceRoleValid(role)
	if err != nil {
		return err
	}
	ctx, err := context.GetCurrentContext()
	if err != nil {
		return err
	}
	if workspaceID == "" {
		workspaceID = ctx.Workspace
	}
	// Get all org users. Setting limit to 1000 for now
	users, err := GetOrgUsers(client)
	if err != nil {
		return err
	}
	userID, email, err := getUserID(email, users, "organization")
	if err != nil {
		return err
	}
	mutateUserInput := astrocore.MutateWorkspaceUserRoleRequest{
		Role: role,
	}
	resp, err := client.MutateWorkspaceUserRoleWithResponse(httpContext.Background(), ctx.Organization, workspaceID, userID, mutateUserInput)
	if err != nil {
		return err
	}
	err = astrocore.NormalizeAPIError(resp.HTTPResponse, resp.Body)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "The user %s was successfully added to the workspace with the role %s\n", email, role)
	return nil
}

func UpdateWorkspaceUserRole(email, role, workspaceID string, out io.Writer, client astrocore.CoreClient) error {
	err := IsWorkspaceRoleValid(role)
	if err != nil {
		return err
	}
	ctx, err := context.GetCurrentContext()
	if err != nil {
		return err
	}
	if workspaceID == "" {
		workspaceID = ctx.Workspace
	}
	// Get all org users. Setting limit to 1000 for now
	users, err := GetWorkspaceUsers(client, workspaceID, userPagnationLimit)
	if err != nil {
		return err
	}
	userID, email, err := getUserID(email, users, "workspace")
	if err != nil {
		return err
	}
	mutateUserInput := astrocore.MutateWorkspaceUserRoleRequest{
		Role: role,
	}
	fmt.Println("workspace: " + workspaceID)
	resp, err := client.MutateWorkspaceUserRoleWithResponse(httpContext.Background(), ctx.Organization, workspaceID, userID, mutateUserInput)
	if err != nil {
		fmt.Println("error in MutateWorkspaceUserRoleWithResponse")
		return err
	}
	err = astrocore.NormalizeAPIError(resp.HTTPResponse, resp.Body)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "The workspace user %s role was successfully updated to %s\n", email, role)
	return nil
}

// IsWorkspaceRoleValid checks if the requested role is valid
// If the role is valid, it returns nil
// error ErrInvalidWorkspaceRole is returned if the role is not valid
func IsWorkspaceRoleValid(role string) error {
	validRoles := []string{"WORKSPACE_MEMBER", "WORKSPACE_AUTHOR", "WORKSPACE_OPERATOR", "WORKSPACE_OWNER"}
	for _, validRole := range validRoles {
		if role == validRole {
			return nil
		}
	}
	return ErrInvalidWorkspaceRole
}

// IsOrgnaizationRoleValid checks if the requested role is valid
// If the role is valid, it returns nil
// error ErrInvalidOrgnaizationRole is returned if the role is not valid
func IsOrganizationRoleValid(role string) error {
	validRoles := []string{"ORGANIZATION_MEMBER", "ORGANIZATION_BILLING_ADMIN", "ORGANIZATION_OWNER"}
	for _, validRole := range validRoles {
		if role == validRole {
			return nil
		}
	}
	return ErrInvalidOrganizationRole
}

// Returns a list of all of a worksapces users
func GetWorkspaceUsers(client astrocore.CoreClient, workspaceID string, limit int) ([]astrocore.User, error) {
	offset := 0
	var users []astrocore.User

	ctx, err := context.GetCurrentContext()
	if err != nil {
		return nil, err
	}

	if workspaceID == "" {
		workspaceID = ctx.Workspace
	}
	for {
		resp, err := client.ListWorkspaceUsersWithResponse(httpContext.Background(), ctx.Organization, workspaceID, &astrocore.ListWorkspaceUsersParams{
			Offset: &offset,
			Limit:  &limit,
		})
		if err != nil {
			return nil, err
		}
		err = astrocore.NormalizeAPIError(resp.HTTPResponse, resp.Body)
		if err != nil {
			return nil, err
		}
		users = append(users, resp.JSON200.Users...)

		if resp.JSON200.TotalCount <= offset {
			break
		}

		offset += limit
	}

	return users, nil
}

// Prints a list of all of a workspaces users
//
//nolint:dupl
func ListWorkspaceUsers(out io.Writer, client astrocore.CoreClient, workspaceID string) error {
	table := printutil.Table{
		Padding:        []int{30, 50, 10, 50, 10, 10, 10},
		DynamicPadding: true,
		Header:         []string{"FULLNAME", "EMAIL", "ID", "WORKSPACE ROLE", "CREATE DATE"},
	}
	users, err := GetWorkspaceUsers(client, workspaceID, userPagnationLimit)
	if err != nil {
		return err
	}

	for i := range users {
		table.AddRow([]string{
			users[i].FullName,
			users[i].Username,
			users[i].Id,
			*users[i].WorkspaceRole,
			users[i].CreatedAt.Format(time.RFC3339),
		}, false)
	}

	table.Print(out)
	return nil
}

func RemoveWorkspaceUser(email, workspaceID string, out io.Writer, client astrocore.CoreClient) error {
	ctx, err := context.GetCurrentContext()
	if err != nil {
		return err
	}
	if workspaceID == "" {
		workspaceID = ctx.Workspace
	}
	// Get all org users. Setting limit to 1000 for now
	users, err := GetWorkspaceUsers(client, workspaceID, userPagnationLimit)
	if err != nil {
		return err
	}
	userID, email, err := getUserID(email, users, "workspace")
	if err != nil {
		return err
	}
	resp, err := client.DeleteWorkspaceUserWithResponse(httpContext.Background(), ctx.Organization, workspaceID, userID)
	if err != nil {
		return err
	}
	err = astrocore.NormalizeAPIError(resp.HTTPResponse, resp.Body)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "The user %s was successfully removed from the workspace\n", email)
	return nil
}

func getUserID(email string, users []astrocore.User, roleEntity string) (userID, newEmail string, err error) {
	if email == "" {
		user, err := SelectUser(users, roleEntity)
		userID = user.Id
		email = user.Username
		if err != nil {
			return "", email, err
		}
	} else {
		for i := range users {
			if users[i].Username == email {
				userID = users[i].Id
			}
		}
	}
	if userID == "" {
		return userID, email, ErrUserNotFound
	}
	return userID, email, nil
}

func GetUser(client astrocore.CoreClient, userID string) (user astrocore.User, err error) {
	ctx, err := context.GetCurrentContext()
	if err != nil {
		return user, err
	}

	resp, err := client.GetUserWithResponse(httpContext.Background(), ctx.Organization, userID)
	if err != nil {
		return user, err
	}
	err = astrocore.NormalizeAPIError(resp.HTTPResponse, resp.Body)
	if err != nil {
		return user, err
	}

	user = *resp.JSON200

	return user, nil
}

func AddDeploymentUser(email, role, deploymentID string, out io.Writer, client astrocore.CoreClient) error {
	ctx, err := context.GetCurrentContext()
	if err != nil {
		return err
	}

	// Get all org users. Setting limit to 1000 for now
	users, err := GetOrgUsers(client)
	if err != nil {
		return err
	}
	userID, email, err := getUserID(email, users, "organization")
	if err != nil {
		return err
	}
	mutateUserInput := astrocore.MutateDeploymentUserRoleRequest{
		Role: role,
	}
	resp, err := client.MutateDeploymentUserRoleWithResponse(httpContext.Background(), ctx.Organization, deploymentID, userID, mutateUserInput)
	if err != nil {
		return err
	}
	err = astrocore.NormalizeAPIError(resp.HTTPResponse, resp.Body)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "The user %s was successfully added to the deployment with the role %s\n", email, role)
	return nil
}

func UpdateDeploymentUserRole(email, role, deploymentID string, out io.Writer, client astrocore.CoreClient) error {
	ctx, err := context.GetCurrentContext()
	if err != nil {
		return err
	}

	// Get all org users. Setting limit to 1000 for now
	users, err := GetDeploymentUsers(client, deploymentID, userPagnationLimit)
	if err != nil {
		return err
	}
	userID, email, err := getUserID(email, users, "deployment")
	if err != nil {
		return err
	}
	mutateUserInput := astrocore.MutateDeploymentUserRoleRequest{
		Role: role,
	}
	resp, err := client.MutateDeploymentUserRoleWithResponse(httpContext.Background(), ctx.Organization, deploymentID, userID, mutateUserInput)
	if err != nil {
		fmt.Println("error in MutateDeploymentUserRoleWithResponse")
		return err
	}
	err = astrocore.NormalizeAPIError(resp.HTTPResponse, resp.Body)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "The deployment user %s role was successfully updated to %s\n", email, role)
	return nil
}

func RemoveDeploymentUser(email, deploymentID string, out io.Writer, client astrocore.CoreClient) error {
	ctx, err := context.GetCurrentContext()
	if err != nil {
		return err
	}

	// Get all org users. Setting limit to 1000 for now
	users, err := GetDeploymentUsers(client, deploymentID, userPagnationLimit)
	if err != nil {
		return err
	}
	userID, email, err := getUserID(email, users, "deployment")
	if err != nil {
		return err
	}
	resp, err := client.DeleteDeploymentUserWithResponse(httpContext.Background(), ctx.Organization, deploymentID, userID)
	if err != nil {
		return err
	}
	err = astrocore.NormalizeAPIError(resp.HTTPResponse, resp.Body)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "The user %s was successfully removed from the deployment\n", email)
	return nil
}

// Returns a list of all of a deployments users
func GetDeploymentUsers(client astrocore.CoreClient, deploymentID string, limit int) ([]astrocore.User, error) {
	offset := 0
	var users []astrocore.User

	ctx, err := context.GetCurrentContext()
	if err != nil {
		return nil, err
	}
	includeDeploymentRoles := true
	for {
		resp, err := client.ListDeploymentUsersWithResponse(httpContext.Background(), ctx.Organization, deploymentID, &astrocore.ListDeploymentUsersParams{
			IncludeDeploymentRoles: &includeDeploymentRoles,
			Offset:                 &offset,
			Limit:                  &limit,
		})
		if err != nil {
			return nil, err
		}
		err = astrocore.NormalizeAPIError(resp.HTTPResponse, resp.Body)
		if err != nil {
			return nil, err
		}
		users = append(users, resp.JSON200.Users...)

		if resp.JSON200.TotalCount <= offset {
			break
		}

		offset += limit
	}

	return users, nil
}

// Prints a list of all of an deployments users
//
//nolint:dupl
func ListDeploymentUsers(out io.Writer, client astrocore.CoreClient, deploymentID string) error {
	table := printutil.Table{
		Padding:        []int{30, 50, 10, 50, 10, 10, 10},
		DynamicPadding: true,
		Header:         []string{"FULLNAME", "EMAIL", "ID", "DEPLOYMENT ROLE", "CREATE DATE"},
	}
	users, err := GetDeploymentUsers(client, deploymentID, userPagnationLimit)
	if err != nil {
		return err
	}

	for i := range users {
		var deploymentRole string
		if users[i].DeploymentRole != nil {
			deploymentRole = *users[i].DeploymentRole
		}
		table.AddRow([]string{
			users[i].FullName,
			users[i].Username,
			users[i].Id,
			deploymentRole,
			users[i].CreatedAt.Format(time.RFC3339),
		}, false)
	}

	table.Print(out)
	return nil
}
