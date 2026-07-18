package api

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var ErrUnsafeSVG = errors.New("unsafe SVG")

const svgNamespace = "http://www.w3.org/2000/svg"

var (
	allowedSVGElements = map[string]bool{
		"svg": true, "g": true, "defs": true, "path": true, "rect": true,
		"circle": true, "ellipse": true, "line": true, "polyline": true,
		"polygon": true, "text": true, "tspan": true, "linearGradient": true,
		"radialGradient": true, "stop": true, "use": true, "title": true, "desc": true,
	}
	globalSVGAttributes = map[string]bool{
		"id": true, "class": true, "transform": true, "fill": true, "stroke": true,
		"stroke-width": true, "opacity": true, "fill-opacity": true, "stroke-opacity": true,
		"style": true,
	}
	elementSVGAttributes = map[string]map[string]bool{
		"svg":            setOf("viewBox", "width", "height", "preserveAspectRatio"),
		"path":           setOf("d", "pathLength"),
		"rect":           setOf("x", "y", "width", "height", "rx", "ry"),
		"circle":         setOf("cx", "cy", "r"),
		"ellipse":        setOf("cx", "cy", "rx", "ry"),
		"line":           setOf("x1", "y1", "x2", "y2"),
		"polyline":       setOf("points"),
		"polygon":        setOf("points"),
		"text":           setOf("x", "y", "dx", "dy", "font-family", "font-size", "text-anchor", "dominant-baseline"),
		"tspan":          setOf("x", "y", "dx", "dy", "font-family", "font-size", "text-anchor", "dominant-baseline"),
		"linearGradient": setOf("x1", "y1", "x2", "y2", "gradientUnits", "gradientTransform"),
		"radialGradient": setOf("cx", "cy", "r", "fx", "fy", "gradientUnits", "gradientTransform"),
		"stop":           setOf("offset", "stop-color", "stop-opacity"),
		"use":            setOf("href", "x", "y", "width", "height"),
	}
	svgIDPattern        = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_.:-]*$`)
	svgClassPattern     = regexp.MustCompile(`^[A-Za-z0-9_.: -]+$`)
	svgNumericPattern   = regexp.MustCompile(`^[+\-]?(?:\d+(?:\.\d*)?|\.\d+)(?:[eE][+\-]?\d+)?%?$`)
	svgNumberList       = regexp.MustCompile(`^[0-9eE+\-.,%\s]+$`)
	svgPathPattern      = regexp.MustCompile(`^[MmZzLlHhVvCcSsQqTtAa0-9eE+\-.,\s]+$`)
	svgTransformPattern = regexp.MustCompile(`^(?:(?:matrix|translate|scale|rotate|skewX|skewY)\s*\([0-9eE+\-.,\s]+\)\s*)+$`)
	svgFragmentPattern  = regexp.MustCompile(`^#[A-Za-z_][A-Za-z0-9_.:-]*$`)
	svgURLPattern       = regexp.MustCompile(`^url\(#[A-Za-z_][A-Za-z0-9_.:-]*\)$`)
	svgColorPattern     = regexp.MustCompile(`^(?:none|currentColor|transparent|#[0-9A-Fa-f]{3,8}|[A-Za-z]+|rgba?\([0-9.,%\s]+\)|url\(#[A-Za-z_][A-Za-z0-9_.:-]*\))$`)
	svgTextValuePattern = regexp.MustCompile(`^[A-Za-z0-9 _.,'"\-]+$`)
)

func setOf(values ...string) map[string]bool {
	result := make(map[string]bool, len(values))
	for _, value := range values {
		result[value] = true
	}
	return result
}

func unsafeSVG(reason string) error { return fmt.Errorf("%w: %s", ErrUnsafeSVG, reason) }

func sanitizeGeneratedSVGResponse(raw string) (string, error) {
	lower := strings.ToLower(raw)
	if strings.Contains(lower, "<!doctype") || strings.Contains(lower, "<!entity") || strings.Contains(lower, "<?") {
		return "", unsafeSVG("directives are not allowed")
	}
	extracted := extractSVG(raw)
	if extracted == "" {
		return "", unsafeSVG("missing SVG root")
	}
	return SanitizeSVG(extracted)
}

// SanitizeSVG validates and normalizes generated SVG using a strict XML allowlist.
func SanitizeSVG(input string) (string, error) {
	decoder := xml.NewDecoder(strings.NewReader(input))
	decoder.Strict = true
	var output strings.Builder
	encoder := xml.NewEncoder(&output)
	var stack []string
	rootSeen := false

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", unsafeSVG("malformed XML")
		}
		switch typed := token.(type) {
		case xml.StartElement:
			if typed.Name.Space != "" && typed.Name.Space != svgNamespace {
				return "", unsafeSVG("unsupported namespace")
			}
			name := typed.Name.Local
			if !allowedSVGElements[name] {
				return "", unsafeSVG("unsupported element")
			}
			if len(stack) == 0 {
				if rootSeen || name != "svg" {
					return "", unsafeSVG("non-SVG root")
				}
				rootSeen = true
			}
			start, err := sanitizeSVGStart(name, typed.Attr, len(stack) == 0)
			if err != nil {
				return "", err
			}
			if err := encoder.EncodeToken(start); err != nil {
				return "", unsafeSVG("encode failure")
			}
			stack = append(stack, name)
		case xml.EndElement:
			if len(stack) == 0 || stack[len(stack)-1] != typed.Name.Local {
				return "", unsafeSVG("mismatched element")
			}
			name := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if err := encoder.EncodeToken(xml.EndElement{Name: xml.Name{Local: name}}); err != nil {
				return "", unsafeSVG("encode failure")
			}
		case xml.CharData:
			if len(stack) == 0 {
				if strings.TrimSpace(string(typed)) != "" {
					return "", unsafeSVG("content outside root")
				}
				continue
			}
			parent := stack[len(stack)-1]
			if parent != "text" && parent != "tspan" && parent != "title" && parent != "desc" && strings.TrimSpace(string(typed)) != "" {
				return "", unsafeSVG("unexpected text content")
			}
			if err := encoder.EncodeToken(typed); err != nil {
				return "", unsafeSVG("encode failure")
			}
		case xml.Comment:
			// Comments are not required for rendering and are omitted.
		case xml.Directive, xml.ProcInst:
			return "", unsafeSVG("directives are not allowed")
		default:
			return "", unsafeSVG("unsupported XML token")
		}
	}
	if !rootSeen || len(stack) != 0 {
		return "", unsafeSVG("incomplete SVG")
	}
	if err := encoder.Flush(); err != nil {
		return "", unsafeSVG("encode failure")
	}
	return output.String(), nil
}

func sanitizeSVGStart(element string, attributes []xml.Attr, root bool) (xml.StartElement, error) {
	result := xml.StartElement{Name: xml.Name{Local: element}}
	seen := make(map[string]bool, len(attributes)+1)
	if root {
		result.Attr = append(result.Attr, xml.Attr{Name: xml.Name{Local: "xmlns"}, Value: svgNamespace})
		seen["xmlns"] = true
	}
	for _, attribute := range attributes {
		name := attribute.Name.Local
		if attribute.Name.Space == "xmlns" || name == "xmlns" {
			if !root || name != "xmlns" || attribute.Value != svgNamespace {
				return xml.StartElement{}, unsafeSVG("unsupported namespace declaration")
			}
			continue
		}
		if attribute.Name.Space != "" {
			return xml.StartElement{}, unsafeSVG("namespaced attribute")
		}
		if seen[name] {
			return xml.StartElement{}, unsafeSVG("duplicate attribute")
		}
		seen[name] = true
		if !globalSVGAttributes[name] && !elementSVGAttributes[element][name] {
			return xml.StartElement{}, unsafeSVG("unsupported attribute")
		}
		value := strings.TrimSpace(attribute.Value)
		if !safeSVGAttribute(name, value) {
			return xml.StartElement{}, unsafeSVG("unsafe attribute value")
		}
		result.Attr = append(result.Attr, xml.Attr{Name: xml.Name{Local: name}, Value: value})
	}
	return result, nil
}

func safeSVGAttribute(name, value string) bool {
	lower := strings.ToLower(value)
	if strings.Contains(lower, "javascript:") || strings.Contains(lower, "data:") || strings.Contains(lower, "http:") || strings.Contains(lower, "https:") || strings.Contains(lower, "@import") || strings.Contains(lower, "expression(") {
		return false
	}
	switch name {
	case "id":
		return svgIDPattern.MatchString(value)
	case "class":
		return svgClassPattern.MatchString(value)
	case "transform", "gradientTransform":
		return svgTransformPattern.MatchString(value)
	case "fill", "stroke", "stop-color":
		return svgColorPattern.MatchString(value)
	case "href":
		return svgFragmentPattern.MatchString(value)
	case "style":
		return safeSVGStyle(value)
	case "d":
		return svgPathPattern.MatchString(value)
	case "points", "viewBox":
		return svgNumberList.MatchString(value)
	case "preserveAspectRatio":
		return value == "none" || svgTextValuePattern.MatchString(value)
	case "gradientUnits":
		return value == "userSpaceOnUse" || value == "objectBoundingBox"
	case "font-family", "text-anchor", "dominant-baseline":
		return svgTextValuePattern.MatchString(value)
	default:
		return svgNumericPattern.MatchString(value)
	}
}

func safeSVGStyle(style string) bool {
	allowed := map[string]bool{
		"fill": true, "stroke": true, "stroke-width": true, "opacity": true,
		"fill-opacity": true, "stroke-opacity": true, "font-family": true,
		"font-size": true, "text-anchor": true,
	}
	for _, declaration := range strings.Split(style, ";") {
		declaration = strings.TrimSpace(declaration)
		if declaration == "" {
			continue
		}
		property, value, ok := strings.Cut(declaration, ":")
		property, value = strings.TrimSpace(property), strings.TrimSpace(value)
		if !ok || !allowed[property] || !safeSVGAttribute(property, value) {
			return false
		}
	}
	return true
}

func (s *Server) persistGeneratedSVG(campaignID int64, name, input string) (int64, error) {
	sanitized, err := SanitizeSVG(input)
	if err != nil {
		return 0, err
	}
	destination := filepath.Join(s.dataDir, "maps")
	if err := os.MkdirAll(destination, 0o750); err != nil {
		return 0, fmt.Errorf("create generated map directory: %w", err)
	}
	temporary, err := os.CreateTemp(destination, ".generated-map-*.tmp")
	if err != nil {
		return 0, fmt.Errorf("create generated map: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o640); err != nil {
		temporary.Close()
		return 0, fmt.Errorf("set generated map permissions: %w", err)
	}
	if _, err := temporary.WriteString(sanitized); err != nil {
		temporary.Close()
		return 0, fmt.Errorf("write generated map: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return 0, fmt.Errorf("close generated map: %w", err)
	}
	filename := "map_" + randomHex(8) + ".svg"
	finalPath := filepath.Join(destination, filename)
	if err := os.Rename(temporaryPath, finalPath); err != nil {
		return 0, fmt.Errorf("publish generated map: %w", err)
	}
	mapID, err := s.db.CreateMap(campaignID, name, "maps/"+filename)
	if err != nil {
		_ = os.Remove(finalPath)
		return 0, fmt.Errorf("save generated map: %w", err)
	}
	return mapID, nil
}
