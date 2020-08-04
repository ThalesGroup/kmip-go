//go:generate kmipgen -o kmip_2_0_additions_generated.go -i kmip_2_0_additions.json -p kmip20

// Package kmip20 contains definitions from the 2.0 specification.  They should eventually
// be merged into the kmip_1_4_specs.json (and that should be renamed to kmip_2_0_specs.json),
// but I didn't have time to merge them in yet.  Just keeping them parked here until I have time
// to incorporate them.
// TODO: should the different versions of the spec be kept in separate declaration files?  Or should
// the ttlv package add a spec version attribute to registration, so servers/clients can configure which
// spec version they want to use, and ttlv would automatically filter allowed values on that?
package kmip20

import (
	"github.com/gemalto/kmip-go/ttlv"
)

// Register2_0Values registers all the additional definitions from the KMIP 2.0 spec.  The registry
// should already contain the 1.4 definitions.
func Register2_0Values(registry *ttlv.Registry) {
	// register new 2.0 values
	// KMIP 2.0 introduces a tag named "Attribute Reference", whose value is the enumeration of all Tags
	registry.RegisterEnum(TagAttributeReference, registry.Tags())

	// KMIP 2.0 has made the value of the Extension Type tag an enumeration of all type values
	registry.RegisterEnum(ttlv.TagExtensionType, registry.Types())

	RegisterGeneratedDefinitions(registry)
}
