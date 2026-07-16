package schemas

import _ "embed"

//go:embed skillcard-v1.json
var SkillCardV1 []byte

//go:embed skillcard-v1alpha2.json
var SkillCardV1Alpha2 []byte
