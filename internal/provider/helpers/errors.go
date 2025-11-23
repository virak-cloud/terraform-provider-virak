package helpers

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// HandleAPIError adds an error diagnostic for API communication failures
func HandleAPIError(diags *diag.Diagnostics, summary string, err error) {
	diags.AddError(
		summary,
		fmt.Sprintf("An unexpected error occurred while communicating with the Virak Cloud API. Error: %s", err),
	)
}

// HandleValidationError adds an error diagnostic for validation failures
func HandleValidationError(diags *diag.Diagnostics, summary string, detail string) {
	diags.AddError(
		summary,
		detail,
	)
}
