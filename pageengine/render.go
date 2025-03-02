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
func CollectCSS(p *PageElement, styleWriter io.Writer, classMap map[*PageElement]string, components map[string]*PageElement, visited map[string]bool, c echo.Context, routeInternal func(string, echo.Context) (*PageElement, error)) {
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
			externalComponent, err := GetExternalComponent(c, p.Import, routeInternal)
			if err != nil {
				fmt.Fprintf(styleWriter, "/* Error: %s */", err)
				return
			}
			// add external component to the components map:
			components[p.Import] = externalComponent

			if p.Text == "" {
				p.Text = externalComponent.Text
			}
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

func GetExternalComponent(c echo.Context, uri string, routeInternal func(string, echo.Context) (*PageElement, error)) (*PageElement, error) {
	log.Println("External resource needed:", uri)

	// Check if the URI is an internal route
	if strings.HasPrefix(uri, "/") {
		log.Println("Attempting to fetch component internally:", uri)
		pageElement, err := routeInternal(uri, c)
		if err == nil {
			return pageElement, nil
		}
		log.Println("Error fetching component internally:", err)
		return nil, fmt.Errorf("error fetching component internally: %w", err)
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
		log.Println("Error decoding JSON:", err)
		return nil, fmt.Errorf("error decoding JSON from %s: %w", uri, err)
	}

	log.Println("Successfully fetched component:", uri)
	return &component, nil
}

// Stream HTML directly using pre-assigned class names
func (p *PageElement) Render(w io.Writer, components map[string]*PageElement, classMap map[*PageElement]string, visited map[string]bool, previewElementMap map[string]*PageElement, nonce string) {
	if p == nil {
		return
	}

	if previewElementMap != nil {
		if p.Pid == "" {
			fmt.Println("Generating new pid for", p.Type)
			p.Pid = generateRandomClassName(6)
		}
		// Ensustore the original element reference, not the rendered/imported one
		if _, exists := previewElementMap[p.Pid]; !exists {
			previewElementMap[p.Pid] = p // Keep reference to the calling element
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

			// Ensure the correct pid is used
			if p.Pid != "" {
				clonedComponent.Pid = p.Pid
			}

			// Render cloned component
			clonedComponent.Render(w, components, classMap, visited, previewElementMap, nonce)

			delete(visited, p.Import) // Allow reuse in different parts of the page
			// delete the import from components now if it contains the private flag
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
	if p.Type == "style" || p.Type == "script" {
		fmt.Fprintf(w, `<%s nonce="%s"`, p.Type, nonce)
	} else {
		fmt.Fprintf(w, "<%s", p.Type)
	}

	// if previewElementMap != nil {

	// 	// Generate a new pid and add it to the preview element map

	// 	if p.Pid == "" {
	// 		fmt.Println("Generating new pid for", p.Type)
	// 		pid := generateRandomClassName(6)
	// 		p.Pid = pid
	// 		fmt.Fprintf(w, ` pid="%s"`, pid)
	// 		previewElementMap[p.Pid] = p
	// 	} else {
	// 		fmt.Printf("Found existing pid for %s : %s\n", p.Type, p.Pid)
	// 		fmt.Fprintf(w, ` pid="%s"`, p.Pid)
	// 		previewElementMap[p.Pid] = p
	// 	}

	// }

	if previewElementMap != nil {
		fmt.Fprintf(w, ` pid="%s"`, p.Pid)
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

	if p.Text != "" {
		fmt.Fprint(w, p.Text)
	}

	// Recursively render child elements
	for i := range p.Elements {
		p.Elements[i].Render(w, components, classMap, visited, previewElementMap, nonce)
	}

	// Close HTML tag
	fmt.Fprintf(w, "</%s>", p.Type)
}

func generateNonce() string {
	b := make([]byte, 16)
	_, err := cryptoRand.Read(b)
	if err != nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(b)
}

// routeInternal is a function
func RenderPage(pageData Page, components map[string]*PageElement, w io.Writer, c echo.Context, routeInternal func(string, echo.Context) (*PageElement, error), previewElementMap map[string]*PageElement) error {
	// map a pid value to a page element so we can target them in the preview

	fmt.Println("rendering page. previewElementMap enabled:", previewElementMap != nil)

	nonce := generateNonce()
	// if rw, ok := w.(http.ResponseWriter); ok {
	// 	//rw.Header().Set("Content-Security-Policy", fmt.Sprintf("default-src 'self'; style-src 'self' 'nonce-%s';", nonce))
	// 	rw.Header().Set("Content-Security-Policy", fmt.Sprintf("default-src 'self'; style-src 'self' 'nonce-%s'; script-src 'self' 'nonce-%s';", nonce, nonce))
	// }
	// Start streaming HTML immediately
	fmt.Fprint(w, "<!DOCTYPE html><html><head>")

	if previewElementMap != nil {
		// add a javascript link to /static/editor.js
		fmt.Fprintf(w, `<script nonce="%s" src="/static/editor.js"></script>`, nonce)
		fmt.Fprintf(w, `<style nonce="%s">body { border: 2px solid red; }</style>`, nonce)
	}

	// Render `<head>` elements
	for i := range pageData.Head.Elements {
		pageData.Head.Elements[i].Render(w, components, nil, nil, previewElementMap, nonce)
	}

	// Collect and stream CSS
	fmt.Fprintf(w, `<style nonce="%s">`, nonce)
	classMap := make(map[*PageElement]string) // Map to track generated class names
	visited := make(map[string]bool)          // Track visited imports to avoid circular dependencies

	for i := range pageData.Body.Elements {
		CollectCSS(&pageData.Body.Elements[i], w, classMap, components, visited, c, routeInternal)
	}
	fmt.Fprint(w, "</style></head><body>")

	visited = make(map[string]bool) // Reset before rendering HTML
	// Render and stream HTML
	for i := range pageData.Body.Elements {
		pageData.Body.Elements[i].Render(w, components, classMap, visited, previewElementMap, nonce)
	}

	fmt.Fprint(w, "</body></html>")

	// Print preview element map for debugging
	// if previewElementMap != nil {
	// 	fmt.Println("Preview element map:")
	// 	for key, _ := range previewElementMap {
	// 		fmt.Printf("%s, pid: %s\n", previewElementMap[key].Type, key)
	// 	}
	// }

	return nil
}
