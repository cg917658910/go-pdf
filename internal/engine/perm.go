package engine

import (
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func restrictPermissions(ctx *model.Context, noPrint, noCopy bool) {
	if ctx.Encrypt == nil {
		//ctx.Encrypt = &pdfcpu.Encrypt{}
	}
	if noPrint {
		//ctx.Encrypt.Permissions &= ^pdfcpu.PermissionsPrint
	}
	if noCopy {
		//ctx.Encrypt.Permissions &= ^pdfcpu.PermissionsExtract
	}
}
