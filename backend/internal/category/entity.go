package category

import (
	"time"

	"github.com/google/uuid"
)

type Category struct {
	ID        uuid.UUID
	UserID    *uuid.UUID
	Name      string
	Icon      *string
	Color     *string
	IsDefault bool
	CreatedAt time.Time
}

type CreateRequest struct {
	Name  string
	Icon  *string
	Color *string
}

type UpdateRequest struct {
	Name  string
	Icon  *string
	Color *string
}
