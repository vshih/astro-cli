package houston

// CreateWorkspace - create a workspace
func (h ClientImplementation) CreateWorkspace(label, description string) (*Workspace, error) {
	req := Request{
		Query:     WorkspaceCreateRequest,
		Variables: map[string]interface{}{"label": label, "description": description},
	}

	r, err := req.DoWithClient(h.client)
	if err != nil {
		return nil, err
	}

	return r.Data.CreateWorkspace, nil
}

// ListWorkspaces - list workspaces
func (h ClientImplementation) ListWorkspaces() ([]Workspace, error) {
	req := Request{
		Query: WorkspacesGetRequest,
	}

	r, err := req.DoWithClient(h.client)
	if err != nil {
		return nil, err
	}

	return r.Data.GetWorkspaces, nil
}

// DeleteWorkspace - delete a workspace
func (h ClientImplementation) DeleteWorkspace(workspaceID string) (*Workspace, error) {
	req := Request{
		Query:     WorkspaceDeleteRequest,
		Variables: map[string]interface{}{"workspaceId": workspaceID},
	}

	res, err := req.DoWithClient(h.client)
	if err != nil {
		return nil, err
	}

	return res.Data.DeleteWorkspace, nil
}

// GetWorkspace - get a workspace
func (h ClientImplementation) GetWorkspace(workspaceID string) (*Workspace, error) {
	// TODO: CHANGE THIS QUERY TO USE THE RIGHT ONE: GET A SINGLE WORKSPACE
	req := Request{
		Query:     WorkspacesGetRequest,
		Variables: map[string]interface{}{"workspaceId": workspaceID},
	}

	res, err := req.DoWithClient(h.client)
	if err != nil {
		return nil, err
	}

	if len(res.Data.GetWorkspaces) < 1 {
		// return error if no workspace found
		return nil, ErrWorkspaceNotFound{workspaceID: workspaceID}
	}

	return &res.Data.GetWorkspaces[0], nil
}

// UpdateWorkspace - update a workspace
func (h ClientImplementation) UpdateWorkspace(workspaceID string, args map[string]string) (*Workspace, error) {
	req := Request{
		Query:     WorkspaceUpdateRequest,
		Variables: map[string]interface{}{"workspaceId": workspaceID, "payload": args},
	}

	r, err := req.DoWithClient(h.client)
	if err != nil {
		return nil, err
	}

	return r.Data.UpdateWorkspace, nil
}