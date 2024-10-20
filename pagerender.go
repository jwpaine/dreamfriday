package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

// Define the H1 component
type H1Component struct {
	Text          string
	Attributes    map[string]string
	CSSProperties map[string]string
}
type DivComponent struct {
	Children      []Component
	Attributes    map[string]string // Generic map for attributes (e.g., class, id, etc.)
	CSSProperties map[string]string // Generic map for CSS properties
}

type PComponent struct {
	Text string
}

func generateRandomClassName(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	rng := rand.New(rand.NewSource(time.Now().UnixNano())) // Use a new random generator with a unique seed
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rng.Intn(len(letters))]
	}
	return string(b)
}

func GenerateCSS(className string, cssProperties map[string]string) string {
	if len(cssProperties) == 0 {
		fmt.Println("No CSS properties found") // Debugging line
		return ""                              // No CSS to generate
	}

	// Generate CSS from the key-value pairs in the CSSProperties map
	css := fmt.Sprintf(".%s {", className)
	for property, value := range cssProperties {
		fmt.Printf("Adding CSS property: %s: %s\n", property, value) // Debugging line
		css += fmt.Sprintf(" %s: %s;", property, value)
	}
	css += " }"
	fmt.Println("Generated CSS:", css) // Debugging line
	return css
}

func (h *H1Component) Render(ctx context.Context, w io.Writer) error {
	// Use the class name from the Attributes map, or default to 'custom-h1'
	className := h.Attributes["class"]
	if className == "" {
		className = "custom-h1" // Fallback if no class is found
	}

	// Generate and render the CSS block using the generic function
	css := GenerateCSS(className, h.CSSProperties)
	if css != "" {
		_, err := w.Write([]byte(fmt.Sprintf("<style>%s</style>", css)))
		if err != nil {
			return err
		}
	}

	// Render the H1 tag with attributes
	_, err := w.Write([]byte(fmt.Sprintf("<h1 class=\"%s\">%s</h1>", className, h.Text)))
	return err
}

func (p PComponent) Render(ctx context.Context, w io.Writer) error {
	element_p := element_p(p.Text)
	return element_p.Render(ctx, w)
}

func (d *DivComponent) Render(ctx context.Context, w io.Writer) error {
	// Use the class name from the Attributes map, or default to 'custom-div'
	className := d.Attributes["class"]
	if className == "" {
		className = "custom-div" // Fallback if no class is found
	}

	// Generate and render the CSS block using the generic function
	css := GenerateCSS(className, d.CSSProperties)
	if css != "" {
		_, err := w.Write([]byte(fmt.Sprintf("<style>%s</style>", css)))
		if err != nil {
			return err
		}
	}

	// Render the opening <div> tag
	var attrs []string
	for attr, value := range d.Attributes {
		attrs = append(attrs, fmt.Sprintf("%s=\"%s\"", attr, value))
	}
	_, err := w.Write([]byte(fmt.Sprintf("<div %s>", strings.Join(attrs, " "))))
	if err != nil {
		return err
	}

	// Render the children
	for _, child := range d.Children {
		err := child.Render(ctx, w)
		if err != nil {
			return err
		}
	}
	_, err = w.Write([]byte("</div>"))
	return err
}

// Component interface for rendering elements
type Component interface {
	Render(ctx context.Context, w io.Writer) error
}

var componentMap = map[string]func(map[string]interface{}, []Component) Component{

	"H1": func(attributes map[string]interface{}, _ []Component) Component {
		// Extract text content
		text, ok := attributes["text"].(string)
		if !ok {
			text = "" // Default to empty string if "text" is not provided
		}

		randomClassName := "h1_" + generateRandomClassName(6) // Generate a 6-character random string
		attr := map[string]string{
			"class": randomClassName,
		}

		// Extract CSS properties from the "style" field in the attributes
		cssProps := map[string]string{}
		if nestedAttributes, ok := attributes["attributes"].(map[string]interface{}); ok {
			if style, ok := nestedAttributes["style"].(map[string]interface{}); ok {
				for key, value := range style {
					if strValue, ok := value.(string); ok {
						cssProps[key] = strValue
					}
				}
			}
		}

		// Return the H1Component with the extracted text, attributes, and CSS
		return &H1Component{
			Text:          text,
			Attributes:    attr,
			CSSProperties: cssProps,
		}
	},
	"P": func(attributes map[string]interface{}, _ []Component) Component {
		// Handle "text" at the top level
		text, ok := attributes["text"].(string)
		if !ok {
			text = "" // Default to empty string if "text" is not provided
		}
		return PComponent{Text: text}
	},
	"Div": func(attributes map[string]interface{}, children []Component) Component {
		randomClassName := "div_" + generateRandomClassName(6) // Generate a 6-character random string
		attr := map[string]string{
			"class": randomClassName,
		}

		// Check if the "attributes" key exists, and extract the nested attributes
		if nestedAttributes, ok := attributes["attributes"].(map[string]interface{}); ok {
			// Now check if the "style" key exists and is of type map inside the nested attributes
			if style, ok := nestedAttributes["style"].(map[string]interface{}); ok {
				fmt.Println("Style found in nested attributes:", style) // Debugging line

				// Extract CSS properties from the "style" field
				cssProps := map[string]string{}
				for key, value := range style {
					if strValue, ok := value.(string); ok {
						fmt.Printf("Adding CSS property: %s: %s\n", key, strValue) // Debugging line
						cssProps[key] = strValue
					}
				}

				// Return a pointer to DivComponent with CSS properties and children
				return &DivComponent{
					Attributes:    attr,
					CSSProperties: cssProps,
					Children:      children,
				}
			}
		}

		// If no style is found, return DivComponent with empty CSSProperties
		fmt.Println("No style found in attributes") // Debugging line
		return &DivComponent{
			Attributes:    attr,
			CSSProperties: map[string]string{},
			Children:      children,
		}
	},
}

// RenderPageContent recursively renders elements from JSON
func RenderPageContent(ctx context.Context, elements []map[string]interface{}) ([]Component, error) {
	var renderedComponents []Component

	// Traverse and render each element
	for _, element := range elements {
		elementType, ok := element["type"].(string)
		if !ok {
			return nil, fmt.Errorf("missing or invalid element type")
		}

		// Parse child elements if they exist
		children := []Component{}
		if childElements, exists := element["elements"].([]interface{}); exists {
			childElementsMap := make([]map[string]interface{}, len(childElements))
			for i, child := range childElements {
				if childMap, ok := child.(map[string]interface{}); ok {
					childElementsMap[i] = childMap
				} else {
					return nil, fmt.Errorf("invalid child element structure")
				}
			}
			var err error
			children, err = RenderPageContent(ctx, childElementsMap)
			if err != nil {
				return nil, err
			}
		}

		// Lookup the component constructor
		if constructor, found := componentMap[elementType]; found {
			component := constructor(element, children)
			renderedComponents = append(renderedComponents, component)
		} else {
			return nil, fmt.Errorf("unknown element type: %s", elementType)
		}
	}

	return renderedComponents, nil
}

// Reusable function to parse JSON, render content, and write to the response
func RenderJSONContent(c echo.Context, jsonContent string) error {
	ctx := c.Request().Context()

	// Parse the JSON content into Go objects
	var pageContent []map[string]interface{}
	err := json.Unmarshal([]byte(jsonContent), &pageContent)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid JSON structure")
	}

	// Call the RenderPageContent function to generate components
	renderedComponents, err := RenderPageContent(ctx, pageContent)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Error rendering page content: "+err.Error())
	}

	// Write the rendered components to the response
	for _, component := range renderedComponents {
		err = component.Render(ctx, c.Response().Writer)
		if err != nil {
			return c.String(http.StatusInternalServerError, "Error rendering component: "+err.Error())
		}
	}

	return nil
}
