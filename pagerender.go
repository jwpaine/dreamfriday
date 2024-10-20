package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/labstack/echo/v4"
)

// Define the H1 component
type H1Component struct {
	Text string
}

func (h H1Component) Render(ctx context.Context, w io.Writer) error {
	_, err := w.Write([]byte(fmt.Sprintf("<h1>%s</h1>", h.Text)))
	return err
}

// Define the P component
type PComponent struct {
	Text string
}

func (p PComponent) Render(ctx context.Context, w io.Writer) error {
	_, err := w.Write([]byte(fmt.Sprintf("<p>%s</p>", p.Text)))
	return err
}

// Define the Div component
type DivComponent struct {
	Children []Component
}

func (d DivComponent) Render(ctx context.Context, w io.Writer) error {
	_, err := w.Write([]byte("<div>"))
	if err != nil {
		return err
	}
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

// ComponentMap maps element types to their component constructors
var componentMap = map[string]func(map[string]interface{}, []Component) Component{
	"H1": func(attributes map[string]interface{}, _ []Component) Component {
		return H1Component{Text: attributes["text"].(string)}
	},
	"P": func(attributes map[string]interface{}, _ []Component) Component {
		return PComponent{Text: attributes["text"].(string)}
	},
	"Div": func(attributes map[string]interface{}, children []Component) Component {
		return DivComponent{Children: children}
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
