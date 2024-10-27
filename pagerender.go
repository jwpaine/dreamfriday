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
	MediaQueries  map[string]map[string]map[string]string
}
type DivComponent struct {
	Children      []Component
	Attributes    map[string]string // Generic map for attributes (e.g., class, id, etc.)
	CSSProperties map[string]string // Generic map for CSS properties
	MediaQueries  map[string]map[string]map[string]string
}

type PComponent struct {
	Text          string
	Attributes    map[string]string
	CSSProperties map[string]string
	MediaQueries  map[string]map[string]map[string]string
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

/*
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
*/

func GenerateCSS(className string, cssProperties map[string]string, mqType string, target string) string {
	// Return empty string if there are no CSS properties
	if len(cssProperties) == 0 {
		return ""
	}

	// Generate the main CSS block from the key-value pairs in the cssProperties map
	var cssContent string
	for property, value := range cssProperties {
		cssContent += fmt.Sprintf(" %s: %s;", property, value)
	}

	// Check if mqType and target are provided for a media query
	if mqType != "" && target != "" {
		// Wrap the CSS in a media query if mqType and target are specified
		return fmt.Sprintf("@media only screen and (%s: %s) { .%s {%s } }", mqType, target, className, cssContent)
	}

	// Generate regular CSS if no media query parameters are provided
	return fmt.Sprintf(".%s {%s }", className, cssContent)
}

/*
func (h *H1Component) Render(ctx context.Context, w io.Writer) error {

	// Generate and render the CSS block using the generic function
	css := GenerateCSS(h.Attributes["class"], h.CSSProperties)

	// Generate media query CSS
	var mediaCSS []string
	for mqType, mqTargets := range h.MediaQueries {
		// mqType is expected to be something like "min-width"
		fmt.Println("Media Query Type:", mqType)
		fmt.Println("Media Query Targets:", mqTargets)
		for target, styles := range mqTargets {
			fmt.Println("Target:", target)
			fmt.Println("Styles:", styles)
			// Generate CSS for the specific styles under this media query
			// queryCSS := GenerateCSS(h.Attributes["class"], styles)
			// if queryCSS != "" {
			// 	mediaCSS = append(mediaCSS, fmt.Sprintf("@media (%s: %s) { .%s { %s } }", mqType, target, h.Attributes["class"], queryCSS))
			// }
		}
	}

	if css != "" {
		_, err := w.Write([]byte(fmt.Sprintf("<style>%s %s</style>", css, strings.Join(mediaCSS, " "))))
		if err != nil {
			return err
		}
	}

	var attrs []string
	for attr, value := range h.Attributes {
		attrs = append(attrs, fmt.Sprintf("%s=\"%s\"", attr, value))
	}
	_, err := w.Write([]byte(fmt.Sprintf("<h1 %s >%s</h1>", strings.Join(attrs, " "), h.Text)))
	return err
} */

func (h *H1Component) Render(ctx context.Context, w io.Writer) error {
	// Generate base CSS for non-media query styles
	baseCSS := GenerateCSS(h.Attributes["class"], h.CSSProperties, "", "")

	// Generate media query CSS
	var mediaCSS []string
	for mqType, mqTargets := range h.MediaQueries {
		for target, styles := range mqTargets {
			queryCSS := GenerateCSS(h.Attributes["class"], styles, mqType, target)
			if queryCSS != "" {
				mediaCSS = append(mediaCSS, queryCSS)
			}
		}
	}

	// Render the complete CSS block
	if baseCSS != "" || len(mediaCSS) > 0 {
		_, err := w.Write([]byte(fmt.Sprintf("<style>%s %s</style>", baseCSS, strings.Join(mediaCSS, " "))))
		if err != nil {
			return err
		}
	}

	// Render the H1 element with its attributes
	var attrs []string
	for attr, value := range h.Attributes {
		attrs = append(attrs, fmt.Sprintf("%s=\"%s\"", attr, value))
	}
	_, err := w.Write([]byte(fmt.Sprintf("<h1 %s>%s</h1>", strings.Join(attrs, " "), h.Text)))
	return err
}

func (p PComponent) Render(ctx context.Context, w io.Writer) error {
	baseCSS := GenerateCSS(p.Attributes["class"], p.CSSProperties, "", "")

	// Generate media query CSS
	var mediaCSS []string
	for mqType, mqTargets := range p.MediaQueries {
		for target, styles := range mqTargets {
			queryCSS := GenerateCSS(p.Attributes["class"], styles, mqType, target)
			if queryCSS != "" {
				mediaCSS = append(mediaCSS, queryCSS)
			}
		}
	}

	// Render the complete CSS block
	if baseCSS != "" || len(mediaCSS) > 0 {
		_, err := w.Write([]byte(fmt.Sprintf("<style>%s %s</style>", baseCSS, strings.Join(mediaCSS, " "))))
		if err != nil {
			return err
		}
	}

	// Render the H1 tag with attributes
	// generate attributes:
	var attrs []string
	for attr, value := range p.Attributes {
		attrs = append(attrs, fmt.Sprintf("%s=\"%s\"", attr, value))
	}
	_, err := w.Write([]byte(fmt.Sprintf("<p %s >%s</p>", strings.Join(attrs, " "), p.Text)))
	return err
}

func (d *DivComponent) Render(ctx context.Context, w io.Writer) error {
	baseCSS := GenerateCSS(d.Attributes["class"], d.CSSProperties, "", "")

	// Generate media query CSS
	var mediaCSS []string
	for mqType, mqTargets := range d.MediaQueries {
		for target, styles := range mqTargets {
			queryCSS := GenerateCSS(d.Attributes["class"], styles, mqType, target)
			if queryCSS != "" {
				mediaCSS = append(mediaCSS, queryCSS)
			}
		}
	}

	// Render the complete CSS block
	if baseCSS != "" || len(mediaCSS) > 0 {
		_, err := w.Write([]byte(fmt.Sprintf("<style>%s %s</style>", baseCSS, strings.Join(mediaCSS, " "))))
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
	/*
		"H1": func(element Models.PageElement, _ []Component) Component {
			// Extract text content from the PageElement
			text := element.Text
			// set element ID
			elementID := element.Attributes.ID
			attr := map[string]string{}
			if elementID != "" {
				attr["id"] = elementID
			}
			// Extract CSS properties from the "style" field in the attributes
			cssProps := map[string]string{}
			for key, value := range element.Attributes.Style {
				if strValue, ok := value.(string); ok {
					cssProps[key] = strValue
				}
			}
			// Generate a random class name if there are CSS properties
			if len(cssProps) > 0 {
				attr["class"] = "H1_" + generateRandomClassName(6)
			}
			// if link set, include hx-get attribute:
			link := element.Link
			if link != "" {
				attr["onclick"] = "window.location.href='" + link + "'"
			}
			// Return the H1Component with the extracted text, attributes, and CSS
			return &H1Component{
				Text:          text,
				Attributes:    attr,
				CSSProperties: cssProps,
			}
		}, */

	"H1": func(element Models.PageElement, _ []Component) Component {
		// Extract text content from the PageElement
		text := element.Text
		// Set element ID
		elementID := element.Attributes.ID
		attr := map[string]string{}
		if elementID != "" {
			attr["id"] = elementID
		}

		// Extract CSS properties from the "style" field in the attributes
		cssProps := map[string]string{}
		mediaQueries := map[string]map[string]map[string]string{}

		for key, value := range element.Attributes.Style {
			switch v := value.(type) {
			case string:
				// Direct CSS property
				cssProps[key] = v
			case map[string]interface{}:
				// Handle nested media queries
				if key == "media" {
					// Iterate through each media query type (e.g., min-width)
					for mqType, mqSettings := range v {
						// Initialize inner map for each media query type if not already present
						if _, exists := mediaQueries[mqType]; !exists {
							mediaQueries[mqType] = map[string]map[string]string{}
						}

						// Check if mqSettings is a nested map (e.g., map[800px:map[font-size:50px]])
						if mqMap, ok := mqSettings.(map[string]interface{}); ok {
							for target, styleMap := range mqMap {
								if styleMap, ok := styleMap.(map[string]interface{}); ok {
									// Collect each style property for the target (e.g., font-size: 50px)
									queryProps := map[string]string{}
									for prop, val := range styleMap {
										if styleValue, ok := val.(string); ok {
											queryProps[prop] = styleValue
										}
									}
									// Store the styles under the target within the media query type
									mediaQueries[mqType][target] = queryProps
								}
							}
						}
					}
				}
			}
		}

		// Generate a random class name if there are CSS properties
		if len(cssProps) > 0 || len(mediaQueries) > 0 {
			attr["class"] = "H1_" + generateRandomClassName(6)
		}

		// If link is set, include hx-get attribute
		link := element.Link
		if link != "" {
			attr["onclick"] = "window.location.href='" + link + "'"
		}

		// Return the H1Component with the extracted text, attributes, and CSS
		return &H1Component{
			Text:          text,
			Attributes:    attr,
			CSSProperties: cssProps,
			MediaQueries:  mediaQueries,
		}
	},

	"P": func(element Models.PageElement, _ []Component) Component {

		// Extract text content from the PageElement
		text := element.Text
		// Set element ID
		elementID := element.Attributes.ID
		attr := map[string]string{}
		if elementID != "" {
			attr["id"] = elementID
		}

		// Extract CSS properties from the "style" field in the attributes
		cssProps := map[string]string{}
		mediaQueries := map[string]map[string]map[string]string{}

		for key, value := range element.Attributes.Style {
			switch v := value.(type) {
			case string:
				// Direct CSS property
				cssProps[key] = v
			case map[string]interface{}:
				// Handle nested media queries
				if key == "media" {
					// Iterate through each media query type (e.g., min-width)
					for mqType, mqSettings := range v {
						// Initialize inner map for each media query type if not already present
						if _, exists := mediaQueries[mqType]; !exists {
							mediaQueries[mqType] = map[string]map[string]string{}
						}

						// Check if mqSettings is a nested map (e.g., map[800px:map[font-size:50px]])
						if mqMap, ok := mqSettings.(map[string]interface{}); ok {
							for target, styleMap := range mqMap {
								if styleMap, ok := styleMap.(map[string]interface{}); ok {
									// Collect each style property for the target (e.g., font-size: 50px)
									queryProps := map[string]string{}
									for prop, val := range styleMap {
										if styleValue, ok := val.(string); ok {
											queryProps[prop] = styleValue
										}
									}
									// Store the styles under the target within the media query type
									mediaQueries[mqType][target] = queryProps
								}
							}
						}
					}
				}
			}
		}

		// Generate a random class name if there are CSS properties
		if len(cssProps) > 0 || len(mediaQueries) > 0 {
			attr["class"] = "P_" + generateRandomClassName(6)
		}

		// If link is set, include hx-get attribute
		link := element.Link
		if link != "" {
			attr["onclick"] = "window.location.href='" + link + "'"
		}

		// Return the H1Component with the extracted text, attributes, and CSS
		return &PComponent{
			Text:          text,
			Attributes:    attr,
			CSSProperties: cssProps,
			MediaQueries:  mediaQueries,
		}
	},

	"Div": func(element Models.PageElement, children []Component) Component {

		// @TODO: create a helper function to set all of this for each element type:
		// set element ID
		elementID := element.Attributes.ID
		attr := map[string]string{}
		if elementID != "" {
			attr["id"] = elementID
		}
		// Extract CSS properties from the "style" field in the attributes
		cssProps := map[string]string{}
		mediaQueries := map[string]map[string]map[string]string{}

		for key, value := range element.Attributes.Style {
			switch v := value.(type) {
			case string:
				// Direct CSS property
				cssProps[key] = v
			case map[string]interface{}:
				// Handle nested media queries
				if key == "media" {
					// Iterate through each media query type (e.g., min-width)
					for mqType, mqSettings := range v {
						// Initialize inner map for each media query type if not already present
						if _, exists := mediaQueries[mqType]; !exists {
							mediaQueries[mqType] = map[string]map[string]string{}
						}

						// Check if mqSettings is a nested map (e.g., map[800px:map[font-size:50px]])
						if mqMap, ok := mqSettings.(map[string]interface{}); ok {
							for target, styleMap := range mqMap {
								if styleMap, ok := styleMap.(map[string]interface{}); ok {
									// Collect each style property for the target (e.g., font-size: 50px)
									queryProps := map[string]string{}
									for prop, val := range styleMap {
										if styleValue, ok := val.(string); ok {
											queryProps[prop] = styleValue
										}
									}
									// Store the styles under the target within the media query type
									mediaQueries[mqType][target] = queryProps
								}
							}
						}
					}
				}
			}
		}
		// Generate a random class name if there are CSS properties
		if len(cssProps) > 0 || len(mediaQueries) > 0 {
			attr["class"] = "Div_" + generateRandomClassName(6)
		}
		// if link set, include hx-get attribute:
		link := element.Link
		if link != "" {
			attr["onclick"] = "window.location.href='" + link + "'"
		}

		return &DivComponent{
			Attributes:    attr,
			CSSProperties: cssProps,
			Children:      children,
			MediaQueries:  mediaQueries,
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
	globalDefaults := `
		<meta name="viewport" content="width=device-width, initial-scale=1">
		<script src="/static/htmx.min.js"></script>

		<style>
			body, p, h1 {
				margin: 0;
				padding: 0;
			}
			@font-face { 
				font-family: open-sans-regular; 
				src: url('/static/font/OpenSans-Regular.ttf') format('truetype');
			}
			@font-face { 
				font-family: open-sans-bold; 
				src: url('/static/font/OpenSans_Bold.ttf') format('truetype');
			}
			html {
				font-size: calc(14px + 0.5vw);
			}
		</style>
	`
	renderedHTML.WriteString("<head>\n")
	renderedHTML.WriteString(globalDefaults)
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
