package storage

import "coordinator/internal/utility"

var (
	ErrJobNotFoundByID     = utility.NewNotFoundError("job with given id doesn't exist")
	ErrRangeNotFoundByID   = utility.NewNotFoundError("range with given id doesn't exist")
	ErrUserNotFoundByEmail = utility.NewNotFoundError("user with the given email doesn't exist")
	ErrUserNotFoundByID    = utility.NewNotFoundError("user with the given id doesn't exist")
	ErrEmailIsTaken        = utility.NewConflictError("user with this email already exists")
	ErrVideoNotFoundByID   = utility.NewNotFoundError("video with given id doesn't exist")
	ErrWorkerNotFoundByID  = utility.NewNotFoundError("worker with given id doesn't exist")
)
