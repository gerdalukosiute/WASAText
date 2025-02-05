package reqcontext

import (
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// RequestContext holds useful data for each request
type RequestContext struct {
	ReqUUID uuid.UUID
	Logger  logrus.FieldLogger
}
