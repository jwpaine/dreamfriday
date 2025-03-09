package pageengine

import (
	cryptoRand "crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

// Map of self-closing tags
var selfClosingTags = map[string]bool{
	"area": true, "base": true, "br": true, "col": true, "embed": true, "hr": true,
	"img": true, "input": true, "link": true, "meta": true, "param": true, "source": true, "track": true, "wbr": true,
}

func generateNonce() string {
	b := make([]byte, 16)
	_, err := cryptoRand.Read(b)
	if err != nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(b)
}

// Generates a random class name
func generateRandomClassName(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rng.Intn(len(letters))]
	}
	return string(b)
}

// Recursive function that collects CSS first and assigns class names
func (pe *PageEngine) CollectCSS(p *PageElement, classMap map[*PageElement]string, visited map[string]bool, routeInternal func(string, echo.Context) (*PageElement, error)) {
	if p == nil {
		return
	}

	// If this element is an imported component, retrieve and process it
	if p.Import != "" {
		visitKey := fmt.Sprintf("%p-%s", p, p.Import) // Unique per instance
		if visited[visitKey] {
			return
		}
		visited[visitKey] = true

		// if external import, fetch the component and add it to the local components map
		// target both http/s:// and / internal routes
		fmt.Println("Import found:", p.Import)

		fmt.Println(p)

		if strings.Contains(p.Import, "/") {
			externalComponent, err := pe.GetExternalComponent(p.Import, routeInternal)
			if err != nil {
				fmt.Fprintf(pe.writer, "/* Error: %s */", err)
				return
			}
			// add external component to the components map:
			pe.components[p.Import] = externalComponent

			if p.Text == "" {
				p.Text = externalComponent.Text
			}
		}

		// now we treat internal and external imports the same way

		if importedComponent, exists := pe.components[p.Import]; exists {
			// Clone the imported component before modification
			clonedComponent := *importedComponent
			clonedComponent.Style = make(map[string]string)

			// Copy original styles
			for key, value := range importedComponent.Style {
				clonedComponent.Style[key] = value
			}
			// Apply local styles
			for key, value := range p.Style {
				clonedComponent.Style[key] = value
			}

			// Process the cloned component
			pe.CollectCSS(&clonedComponent, classMap, visited, routeInternal)

			// Assign the cloned component’s class name to the referencing element
			if className, ok := classMap[&clonedComponent]; ok {
				classMap[p] = className
			}
		}

		return // Don't generate CSS for the referencing import itself
	}

	// Generate and store the class name once
	className := fmt.Sprintf("%s_%s", p.Type, generateRandomClassName(6))
	classMap[p] = className // Store in map

	// Stream CSS immediately using stored class name
	if len(p.Style) > 0 {
		pe.GenerateCSS(className, p.Style)
	}

	// Recursively collect CSS for child elements
	for i := range p.Elements {
		pe.CollectCSS(&p.Elements[i], classMap, visited, routeInternal)
	}
}

// Generate and write CSS styles directly to `styleWriter`
func (pe *PageEngine) GenerateCSS(className string, css map[string]string) {

	if len(css) == 0 {
		return
	}
	fmt.Fprintf(pe.writer, ".%s {", className)
	for key, value := range css {
		fmt.Fprintf(pe.writer, " %s: %s;", key, value)
	}
	fmt.Fprint(pe.writer, " }") // Close the CSS rule
}

func (pe *PageEngine) GetExternalComponent(uri string, routeInternal func(string, echo.Context) (*PageElement, error)) (*PageElement, error) {
	log.Println("External resource needed:", uri)

	// Check if the URI is an internal route
	if strings.HasPrefix(uri, "/") {
		log.Println("Attempting to fetch component internally:", uri)
		pageElement, err := routeInternal(uri, pe.ctx)
		if err == nil {
			return pageElement, nil
		}
		log.Println("Error fetching component internally:", err)
		return nil, fmt.Errorf("error fetching component internally: %w", err)
	}

	log.Println("Attempting to fetch component externally:", uri)

	// Prepare external HTTP request
	req, err := http.NewRequestWithContext(pe.ctx.Request().Context(), "GET", uri, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for %s: %w", uri, err)
	}

	// Copy headers from the original request
	for key, values := range pe.ctx.Request().Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	// Perform the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error fetching %s: %w", uri, err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response from %s: %w", uri, err)
	}

	// Decode JSON into PageElement
	var component PageElement
	err = json.Unmarshal(body, &component)
	if err != nil {
		log.Println("Error decoding JSON:", err)
		return nil, fmt.Errorf("error decoding JSON from %s: %w", uri, err)
	}

	log.Println("Successfully fetched component:", uri)
	return &component, nil
}

// Stream HTML directly using pre-assigned class names
// Stream HTML directly using pre-assigned class names
func (p *PageElement) RenderElement(pe *PageEngine, classMap map[*PageElement]string, visited map[string]bool, previewElementMap map[string]*PageElement, nonce string) {
	if p == nil {
		return
	}

	// If preview mode, set a PID for each element mapping to the PageElement
	if previewElementMap != nil {
		p.Pid = generateRandomClassName(6) // Assign a unique identifier

		// Store the original element reference
		if _, exists := previewElementMap[p.Pid]; !exists {
			previewElementMap[p.Pid] = p
		}
	}

	// Handle imported components
	if p.Import != "" {
		// Prevent circular dependencies
		visitKey := fmt.Sprintf("%p-%s", p, p.Import) // Unique per instance
		if visited[visitKey] {
			return
		}
		visited[visitKey] = true

		// Handle internal imports
		if importedComponent, exists := pe.components[p.Import]; exists {
			// Deep clone the component to prevent global state pollution
			clonedComponent := *importedComponent
			clonedComponent.Style = make(map[string]string)

			// Copy original styles
			for key, value := range importedComponent.Style {
				clonedComponent.Style[key] = value
			}
			// Apply instance-specific modifications
			for key, value := range p.Style {
				clonedComponent.Style[key] = value
			}

			// Ensure cloned component has an attributes map
			if clonedComponent.Attributes == nil {
				clonedComponent.Attributes = make(map[string]string)
			}

			// Copy locally defined attributes to the cloned component
			for key, value := range p.Attributes {
				clonedComponent.Attributes[key] = value
			}

			// Override text if specified
			if p.Text != "" {
				clonedComponent.Text = p.Text
			}

			// Ensure correct class name is used
			if className, ok := classMap[p]; ok {
				clonedComponent.Attributes["class"] = className
			}

			// Ensure correct PID is used
			if p.Pid != "" {
				clonedComponent.Pid = p.Pid
			}

			// Render cloned component
			clonedComponent.RenderElement(pe, classMap, visited, previewElementMap, nonce)

			// Allow reuse in different parts of the page by removing visit lock
			delete(visited, p.Import)

			// If component is marked as private, delete after use
			if p.Private {
				fmt.Println("Private component found, deleting from components")
				delete(pe.components, p.Import)
			}

			return
		}
	}

	// Retrieve stored class name (if exists)
	className, hasClass := classMap[p]

	// Open HTML tag
	if p.Type == "style" || p.Type == "script" {
		fmt.Fprintf(pe.writer, `<%s nonce="%s"`, p.Type, nonce)
	} else {
		fmt.Fprintf(pe.writer, "<%s", p.Type)
	}

	if previewElementMap != nil {
		fmt.Fprintf(pe.writer, ` pid="%s"`, p.Pid)
	}

	// Process attributes
	var customClass string
	for key, value := range p.Attributes {
		if key == "class" {
			customClass = value
			continue
		}
		fmt.Fprintf(pe.writer, ` %s="%s"`, key, value)
	}

	// Assign class names correctly
	if hasClass || customClass != "" {
		fmt.Fprint(pe.writer, ` class="`)
		if hasClass {
			fmt.Fprint(pe.writer, className)
			if customClass != "" {
				fmt.Fprint(pe.writer, " ")
			}
		}
		if customClass != "" {
			fmt.Fprint(pe.writer, customClass)
		}
		fmt.Fprint(pe.writer, `"`)
	}

	// Handle self-closing tags
	if selfClosingTags[p.Type] {
		fmt.Fprint(pe.writer, " />")
		return
	}

	fmt.Fprint(pe.writer, ">")

	// Print text content if present
	if p.Text != "" {
		fmt.Fprint(pe.writer, p.Text)
	}

	// Recursively render child elements
	for i := range p.Elements {
		p.Elements[i].RenderElement(pe, classMap, visited, previewElementMap, nonce)
	}

	// Close HTML tag
	fmt.Fprintf(pe.writer, "</%s>", p.Type)
}

// routeInternal is a function
func (pe *PageEngine) RenderPage(pageData Page, routeInternal func(string, echo.Context) (*PageElement, error), previewElementMap map[string]*PageElement) error {
	// map a pid value to a page element so we can target them in the preview

	fmt.Println("rendering page. previewElementMap enabled:", previewElementMap != nil)

	nonce := generateNonce()

	// Start streaming HTML immediately
	fmt.Fprint(pe.writer, "<!DOCTYPE html><html><head>")

	if previewElementMap != nil {
		// add a javascript link to /static/editor.js
		fmt.Fprintf(pe.writer, `<script nonce="%s" src="/static/editor.js"></script>`, nonce)
		fmt.Fprintf(pe.writer, `<style nonce="%s">body { border: 2px solid red; }</style>`, nonce)
	}

	// Render `<head>` elements
	for i := range pageData.Head.Elements {
		pageData.Head.Elements[i].RenderElement(pe, nil, nil, previewElementMap, nonce)
	}

	// Collect and stream CSS
	fmt.Fprintf(pe.writer, `<style nonce="%s">`, nonce)
	classMap := make(map[*PageElement]string) // Map to track generated class names
	visited := make(map[string]bool)          // Track visited imports to avoid circular dependencies

	for i := range pageData.Body.Elements {
		pe.CollectCSS(&pageData.Body.Elements[i], classMap, visited, routeInternal)
	}
	fmt.Fprint(pe.writer, "</style></head><body>")

	visited = make(map[string]bool) // Reset before rendering HTML
	// Render and stream HTML
	for i := range pageData.Body.Elements {
		pageData.Body.Elements[i].RenderElement(pe, classMap, visited, previewElementMap, nonce)
	}

	fmt.Fprint(pe.writer, "</body></html>")

	return nil
}

type PageEngine struct {
	ctx        echo.Context
	writer     io.Writer
	components map[string]*PageElement
}

// NewPageEngine initializes an instance with request-specific context
func NewPageEngine(context echo.Context, comps map[string]*PageElement) *PageEngine {
	return &PageEngine{
		ctx:        context,
		writer:     context.Response().Writer,
		components: comps,
	}
}
