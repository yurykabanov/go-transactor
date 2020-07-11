package transactor

import (
	"github.com/opentracing/opentracing-go"
)

func tagSpanWithError(span opentracing.Span, err *error) {
	if *err != nil {
		span.SetTag("error", true).
			LogKV(
				"event", "error",
				"error", (*err).Error(),
			)
	}
}
