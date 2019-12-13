// Package kmip20 contains definitions from the 2.0 specification.  They should eventually
// be merged into the kmip_1_4_specs.json (and that should be renamed to kmip_2_0_specs.json),
// but I didn't have time to merge them in yet.  Just keeping them parked here until I have time
// to incorporate them.
// TODO: should the different versions of the spec be kept in separate declaration files?  Or should
// the ttlv package add a spec version attribute to registration, so servers/clients can configure which
// spec version they want to use, and ttlv would automatically filter allowed values on that?
package kmip20
