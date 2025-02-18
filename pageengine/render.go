package pageengine

import (
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
func CollectCSS(p *PageElement, styleWriter io.Writer, classMap map[*PageElement]string, components map[string]*PageElement, visited map[string]bool, c echo.Context, routeInternal func(string, echo.Context) (interface{}, error)) {
	if p == nil {
		return
	}

	// If this element is an imported component, retrieve and process it
	if p.Import != "" {
		if visited[p.Import] {
			return // Prevent circular dependencies
		}
		visited[p.Import] = true // Mark component as visited

		// if external import, fetch the component and add it to the local components map
		// target both http/s:// and / internal routes
		fmt.Println("Import found:", p.Import)
		if strings.Contains(p.Import, "/") {
			externalComponent, err := GetExternalComponent(c, p.Import, routeInternal)
			if err != nil {
				fmt.Fprintf(styleWriter, "/* Error: %s */", err)
				return
			}
			// add external component to the components map:
			components[p.Import] = externalComponent
		}

		// now we treat internal and external imports the same way

		if importedComponent, exists := components[p.Import]; exists {
			// Ensure CSS is only generated once per imported component
			if _, alreadyProcessed := classMap[importedComponent]; !alreadyProcessed {
				// copy local styles to the imported component
				if importedComponent.Style == nil {
					importedComponent.Style = make(map[string]string)
				}
				for key, value := range p.Style {
					importedComponent.Style[key] = value
				}
				CollectCSS(importedComponent, styleWriter, classMap, components, visited, c, routeInternal)
			}

			// Assign the imported component's class name to the referencing element (`p`)
			if className, ok := classMap[importedComponent]; ok {
				classMap[p] = className // Ensure `p` uses the same class
			}
		}
		return // Don't generate CSS for the referencing import itself
	}

	// Generate and store the class name once
	className := fmt.Sprintf("%s_%s", p.Type, generateRandomClassName(6))
	classMap[p] = className // Store in map

	// Stream CSS immediately using stored class name
	if len(p.Style) > 0 {
		GenerateCSS(className, p.Style, styleWriter)
	}

	// Recursively collect CSS for child elements
	for i := range p.Elements {
		CollectCSS(&p.Elements[i], styleWriter, classMap, components, visited, c, routeInternal)
	}
}

// Generate and write CSS styles directly to `styleWriter`
func GenerateCSS(className string, css map[string]string, styleWriter io.Writer) {
	if len(css) == 0 {
		return
	}
	fmt.Fprintf(styleWriter, ".%s {", className)
	for key, value := range css {
		fmt.Fprintf(styleWriter, " %s: %s;", key, value)
	}
	fmt.Fprint(styleWriter, " }") // Close the CSS rule
}

func GetExternalComponent(c echo.Context, uri string, routeInternal func(string, echo.Context) (interface{}, error)) (*PageElement, error) {
	log.Println("External component needed:", uri)

	// Check if the URI is an internal route
	if strings.HasPrefix(uri, "/") {
		log.Println("Attempting to fetch component internally:", uri)
		pageElement, err := routeInternal(uri, c)
		if err == nil {
			if component, ok := pageElement.(PageElement); ok {
				return &component, nil
			}
			return nil, fmt.Errorf("invalid response type from internal route")
		}
	}

	log.Println("Attempting to fetch component externally:", uri)

	// Prepare external HTTP request
	req, err := http.NewRequestWithContext(c.Request().Context(), "GET", uri, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for %s: %w", uri, err)
	}

	// Copy headers from the original request
	for key, values := range c.Request().Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	// Copy cookies from the original request
	// for _, cookie := range c.Request().Cookies() {
	// 	req.AddCookie(cookie)
	// }

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
		return nil, fmt.Errorf("error decoding JSON from %s: %w", uri, err)
	}

	log.Println("Successfully fetched component:", uri)
	return &component, nil
}

// Stream HTML directly using pre-assigned class names
func (p *PageElement) Render(w io.Writer, components map[string]*PageElement, classMap map[*PageElement]string, visited map[string]bool, previewElementMap map[string]*PageElement) {
	if p == nil {
		return
	}

	// Handle imported components
	if p.Import != "" {
		// Prevent circular dependencies
		if visited[p.Import] {
			return
		}
		visited[p.Import] = true

		// handle internal imports
		if importedComponent, exists := components[p.Import]; exists {
			clonedComponent := *importedComponent // Clone to prevent global state pollution (multiple imports using the same component)

			// Ensure cloned component has an attributes map
			if clonedComponent.Attributes == nil {
				clonedComponent.Attributes = make(map[string]string)
			}

			// Copy locally defined values to the cloned component
			for key, value := range p.Attributes {
				clonedComponent.Attributes[key] = value
			}
			if p.Text != "" {
				clonedComponent.Text = p.Text
			}

			// Ensure the correct class name is used
			if className, ok := classMap[p]; ok {
				clonedComponent.Attributes["class"] = className
			}

			// Render cloned component
			clonedComponent.Render(w, components, classMap, visited, previewElementMap)

			delete(visited, p.Import) // Allow reuse in different parts of the page
			// delete the import from components now if it contains the private flag

			// print imported component for debugging:
			// print p.Private
			if p.Private {
				fmt.Println("Private component found, deleting from components")
				delete(components, p.Import)
			}

			return
		}
	}

	// Retrieve stored class name (if exists)
	className, hasClass := classMap[p]

	// Open HTML tag
	fmt.Fprintf(w, "<%s", p.Type)

	if previewElementMap != nil {
		if p.Pid == "" {
			// Generate a new pid and add it to the preview element map
			// fmt.Println("Generating new pid for:", p.Type)
			if hasClass {
				fmt.Fprintf(w, ` pid="%s"`, className)
				previewElementMap[className] = p
				p.Pid = className
			} else {
				pid := generateRandomClassName(6)
				fmt.Fprintf(w, ` pid="%s"`, pid)
				previewElementMap[pid] = p
				p.Pid = pid
			}
		} else {
			// fmt.Println("pid already exists:", p.Pid)
		}
	}

	// Process attributes
	var customClass string
	for key, value := range p.Attributes {
		if key == "class" {
			customClass = value
			continue
		}
		fmt.Fprintf(w, ` %s="%s"`, key, value)
	}

	// Assign class names correctly
	if hasClass || customClass != "" {
		fmt.Fprint(w, ` class="`)
		if hasClass {
			fmt.Fprint(w, className)
			if customClass != "" {
				fmt.Fprint(w, " ")
			}
		}
		if customClass != "" {
			fmt.Fprint(w, customClass)
		}
		fmt.Fprint(w, `"`)
	}

	// Handle self-closing tags
	if selfClosingTags[p.Type] {
		fmt.Fprint(w, " />")
		return
	}

	fmt.Fprint(w, ">")

	// Print text content if present
	if p.Text != "" {
		fmt.Fprint(w, p.Text)
	}

	// Recursively render child elements
	for i := range p.Elements {
		p.Elements[i].Render(w, components, classMap, visited, previewElementMap)
	}

	// Close HTML tag
	fmt.Fprintf(w, "</%s>", p.Type)
}

// routeInternal is a function
func RenderPage(pageData Page, components map[string]*PageElement, w io.Writer, c echo.Context, routeInternal func(string, echo.Context) (interface{}, error), previewElementMap map[string]*PageElement) error {
	// map a pid value to a page element so we can target them in the preview

	fmt.Println("rendering page. previewElementMap enabled:", previewElementMap != nil)

	// Start streaming HTML immediately
	fmt.Fprint(w, "<!DOCTYPE html><html><head>")

	// Render `<head>` elements
	for i := range pageData.Head.Elements {
		pageData.Head.Elements[i].Render(w, components, nil, nil, previewElementMap)
	}

	// Collect and stream CSS
	fmt.Fprint(w, "<style>")
	classMap := make(map[*PageElement]string) // Map to track generated class names
	visited := make(map[string]bool)          // Track visited imports to avoid circular dependencies

	for i := range pageData.Body.Elements {
		CollectCSS(&pageData.Body.Elements[i], w, classMap, components, visited, c, routeInternal)
	}
	fmt.Fprint(w, "</style></head><body>")

	visited = make(map[string]bool) // Reset before rendering HTML
	// Render and stream HTML
	for i := range pageData.Body.Elements {
		pageData.Body.Elements[i].Render(w, components, classMap, visited, previewElementMap)
	}

	fmt.Fprint(w, "</body></html>")

	// Print preview element map for debugging
	if previewElementMap != nil {
		fmt.Println("Preview element map:")
		for key, _ := range previewElementMap {
			fmt.Printf("%s, pid: %s\n", previewElementMap[key].Type, key)
		}
	}

	return nil
}
