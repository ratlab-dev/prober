package probe

import "context"

type Prober interface {
	Probe(ctx context.Context) error
	MetadataString() string
}
