//go:generate kmipgen -o kmip_2_0_additions_generated.go -i kmip_2_0_additions.json -p kmip20

package kmip20

import (
	"github.com/gemalto/kmip-go/ttlv"
)

func Register2_0Values(registry *ttlv.Registry) {
	// register new 2.0 values
	// KMIP 2.0 introduces a tag named "Attribute Reference", whose value is the enumeration of all Tags
	registry.RegisterEnum(TagAttributeReference, registry.Tags())

	// KMIP 2.0 has made the value of the Extension Type tag an enumeration of all type values
	registry.RegisterEnum(ttlv.TagExtensionType, registry.Types())

	RegisterGeneratedDefinitions(registry)
}
