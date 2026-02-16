package html

import _ "embed"

//go:embed assets/template.html
var templateHTML string

//go:embed assets/styles.css
var stylesCSS string

//go:embed assets/viewer.js
var viewerJS string
