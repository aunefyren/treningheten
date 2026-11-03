package controllers

import (
	"github.com/aunefyren/treningheten/internal/sanitize"
	"github.com/gin-gonic/gin"
)

// bindJSON decodes the request body into target and then applies its `sanitize` tags, so a
// handler only ever sees trimmed, normalised input. Use it instead of ShouldBindJSON; each
// handler keeps its own error response. See docs/conventions.md → Input sanitisation.
func bindJSON(context *gin.Context, target any) error {
	if err := context.ShouldBindJSON(target); err != nil {
		return err
	}
	sanitize.Struct(target)
	return nil
}
