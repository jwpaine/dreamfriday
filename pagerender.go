package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	Models "dreamfriday/models"
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
	Text          string
	Attributes    map[string]string
	CSSProperties map[string]string
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
		// fmt.Println("No CSS properties found") // Debugging line
		return "" // No CSS to generate
	}

	// Generate CSS from the key-value pairs in the CSSProperties map
	css := fmt.Sprintf(".%s {", className)
	for property, value := range cssProperties {
		// fmt.Printf("Adding CSS property: %s: %s\n", property, value) // Debugging line
		css += fmt.Sprintf(" %s: %s;", property, value)
	}
	css += " }"
	// fmt.Println("Generated CSS:", css) // Debugging line
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
	className := p.Attributes["class"]
	if className == "" {
		className = "custom-p" // Fallback if no class is found
	}

	// Generate and render the CSS block using the generic function
	css := GenerateCSS(className, p.CSSProperties)
	if css != "" {
		_, err := w.Write([]byte(fmt.Sprintf("<style>%s</style>", css)))
		if err != nil {
			return err
		}
	}

	// Render the H1 tag with attributes
	_, err := w.Write([]byte(fmt.Sprintf("<p class=\"%s\">%s</p>", className, p.Text)))
	return err
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

var componentMap = map[string]func(Models.PageElement, []Component) Component{

	"H1": func(element Models.PageElement, _ []Component) Component {
		// Extract text content from the PageElement
		text := element.Text

		randomClassName := "h1_" + generateRandomClassName(6) // Generate a 6-character random string
		attr := map[string]string{
			"class": randomClassName,
		}

		// Extract CSS properties from the "style" field in the attributes
		cssProps := map[string]string{}
		for key, value := range element.Attributes.Style {
			if strValue, ok := value.(string); ok {
				cssProps[key] = strValue
			}
		}

		// Debug: Log the extracted CSS properties
		if len(cssProps) == 0 {
			// log.Println("No CSS properties found")
		} else {
			// log.Printf("CSS properties: %+v", cssProps)
		}

		// Return the H1Component with the extracted text, attributes, and CSS
		return &H1Component{
			Text:          text,
			Attributes:    attr,
			CSSProperties: cssProps,
		}
	},

	"P": func(element Models.PageElement, _ []Component) Component {
		// Extract text content from the PageElement
		text := element.Text

		randomClassName := "p_" + generateRandomClassName(6) // Generate a 6-character random string
		attr := map[string]string{
			"class": randomClassName,
		}

		// Extract CSS properties from the "style" field in the attributes
		cssProps := map[string]string{}
		for key, value := range element.Attributes.Style {
			if strValue, ok := value.(string); ok {
				cssProps[key] = strValue
			}
		}

		// Return the PComponent with the extracted text, attributes, and CSS
		return &PComponent{
			Text:          text,
			Attributes:    attr,
			CSSProperties: cssProps,
		}
	},

	"Div": func(element Models.PageElement, children []Component) Component {
		randomClassName := "div_" + generateRandomClassName(6) // Generate a 6-character random string
		attr := map[string]string{
			"class": randomClassName,
		}

		// Extract CSS properties from the "style" field in the attributes
		cssProps := map[string]string{}
		for key, value := range element.Attributes.Style {
			if strValue, ok := value.(string); ok {
				cssProps[key] = strValue
			}
		}

		// Debug: Log the extracted CSS properties
		// if len(cssProps) == 0 {
		// 	log.Println("No CSS properties found")
		// } else {
		// 	log.Printf("CSS properties: %+v", cssProps)
		// }

		// Return a pointer to DivComponent with CSS properties and children
		return &DivComponent{
			Attributes:    attr,
			CSSProperties: cssProps,
			Children:      children,
		}
	},
}

// RenderPageContent recursively renders elements from JSON
func RenderPageContent(ctx context.Context, elements []Models.PageElement) ([]Component, error) {
	var renderedComponents []Component

	// Traverse and render each element
	for _, element := range elements {
		// Retrieve the element type
		elementType := element.Type

		// Parse child elements if they exist
		children := []Component{}
		if len(element.Elements) > 0 {
			// Recursively call RenderPageContent for nested elements
			var err error
			children, err = RenderPageContent(ctx, element.Elements)
			if err != nil {
				return nil, err
			}
		}

		// Lookup the component constructor based on the element type
		if constructor, found := componentMap[elementType]; found {
			component := constructor(element, children)
			renderedComponents = append(renderedComponents, component)
		} else {
			return nil, fmt.Errorf("unknown element type: %s", elementType)
		}
	}

	return renderedComponents, nil
}

func RenderJSONContent(c echo.Context, jsonContent interface{}) error {
	ctx := c.Request().Context()

	// Debug: Ensure jsonContent is not nil
	if jsonContent == nil {
		log.Println("jsonContent is nil")
		return c.String(http.StatusInternalServerError, "No content provided")
	}

	// Debug: Log the type of jsonContent to ensure it's correct
	log.Printf("jsonContent type: %T", jsonContent)

	// Assert that jsonContent is a slice of PageElement
	pageContent, ok := jsonContent.([]Models.PageElement)
	if !ok {
		log.Println("jsonContent is of the wrong type. Expected []PageElement")
		return c.String(http.StatusBadRequest, "Invalid content structure, expected []PageElement")
	}

	// Call the RenderPageContent function to generate components
	renderedComponents, err := RenderPageContent(ctx, pageContent)
	if err != nil {
		log.Println("Error rendering page content:", err)
		return c.String(http.StatusInternalServerError, "Error rendering page content: "+err.Error())
	}

	if len(renderedComponents) == 0 {
		log.Println("No components to render")
		return c.String(http.StatusOK, "No content to render")
	}

	var renderedHTML strings.Builder

	// Always include a script tag in the header
	globalStyling := `
		<style>
			body {
				margin: 0;
				padding: 0;
			}
		</style>
	`
	renderedHTML.WriteString("<head>\n")
	renderedHTML.WriteString(globalStyling)
	renderedHTML.WriteString("\n</head>\n")

	// Write the rendered components to the response
	for _, component := range renderedComponents {
		err = component.Render(ctx, &renderedHTML)
		if err != nil {
			log.Println("Error rendering component:", err)
			return c.String(http.StatusInternalServerError, "Error rendering component: "+err.Error())
		}
	}

	// Output the full HTML content
	log.Println("Rendering HTML content successfully")
	return c.HTML(http.StatusOK, renderedHTML.String())
}
