package repository

import "database/sql"

type All struct {
	User         *UserRepo
	Organization *OrganizationRepo
	Release      *ReleaseRepo
	Mute         *MuteRepo
	Repository   *RepositoryRepo
}

func NewAll(db *sql.DB) *All {
	return &All{
		User:         &UserRepo{db: db},
		Organization: &OrganizationRepo{db: db},
		Release:      &ReleaseRepo{db: db},
		Mute:         &MuteRepo{db: db},
		Repository:   &RepositoryRepo{db: db},
	}
}
