package api

import "go.rtnl.ai/quarterdeck/pkg/store/models"

type Role struct {
	ID    int    `json:"id,omitempty"`
	Title string `json:"title,omitempty"`
}

func (r *Role) Model() (model *models.Role) {
	model = &models.Role{
		ID:    int64(r.ID),
		Title: r.Title,
	}

	return model
}
