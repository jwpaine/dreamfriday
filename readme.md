# Dream Friday

**Dream Friday** 

A tiny, JSON-based CMS built with Go and PostgreSQL. Page endering engine will dynamically construct a component tree by interpreting JSON data stored in PostgreSQL. Upon request, the engine retrieves this data, which defines the topology, type, attributes, and nested elements of each page component, and recursively builds a tree of HTML elements spanning tags like <div>, <p>, and <h1>. Each component generates its own HTML structure and applies CSS styling, with randomized class names to keep styles modular. All styling is aggregated and injected into the document head:

- **Components**: Reusable elements (like buttons or headers) can be defined and re-used anywhere across the site, and specific attributes and properties can be overridden on import.
- **Pages**: Define each page’s layout and content. New page routes are created on the fly and will be immedietly accessible via /{page_name}
- **Attributes and Styling**: Styles and attributes are locally scoped to each element.
- **Media Queries**: Define responsive styling directly in JSON.
- **Preview Mode**: Test mode can be enabled for the site owner by hitting `/preview` while logged in, and then you can navigate around the site in preview mode.

## JSON Example

### Components and Pages

```json
{
  "Components": {
    "button": {
      "elements": [
        {
          "type": "button",
          "attributes": {
            "style": {
              "background": "blue",
              "color": "white",
              "padding": "10px 20px",
              "media": {
                "max-width": {
                  "735px": {
                    "background": "lightblue",
                    "color": "green"
                  }
                }
              }
            }
          },
          "text": "Click Me"
        }
      ]
    }
  },
  "pages": {
    "home": {
      "elements": [
        {
          "type": "h1",
          "text": "Welcome!"
        },
        {
          "import": "button",
          "text": "Get Started",
          "attributes": {
            "style": {
              "background": "green"
            }
          }
        }
      ]
    },
    "about": {
      "elements": [
        {
          "type": "h1",
          "text": "About Us"
        },
        {
          "type": "p",
          "text": "Learn more about us."
        }
      ]
    }
  }
}
```

### Explanation

- **Components**: Define reusable elements like `button` with customizable styling and media queries.

- **Pages**: Define specific page layouts like `home` and `about`. For example:

  - **home**: Uses `button` with a text override to `"Get Started"` and a background color override.

  - **about**: Displays a static heading and paragraph.

### Routing

- **Home Page (`home`)**: Accessible at `/` or `/home`.

- **Named Pages**: For example, the `"about"` page is accessible at `/about`.

### Preview Mode

Dream Friday allows you to preview changes before they are published. You can enable preview mode by accessing the route `/preview`, which will load any saved drafts that haven’t been published.

### Admin Interface

The admin page provides a JSON editor that allows you to manage content and components directly. With the editor, you can:

- **Edit JSON Data**: Modify page structure, styling, and components.
- **Save to Preview**: Save drafts to preview mode without affecting live content.
- **Publish Changes**: Publish saved drafts to go live on the main site.

### Customization

#### Adding New Components

Add reusable components under `"Components"` in the JSON structure. For example, a footer:

```json

"Components": {
  "footer": {
    "elements": [
      {
        "type": "footer",
        "attributes": {
          "style": {
            "background": "#333",
            "color": "white",
            "padding": "10px"
          }
        },
        "text": "© 2024 Dream Friday"
      }
    ]
  }
}
```

#### Adding New Pages

Define new pages under `"pages"` with a unique key. Use `"import"` to include reusable components.

Example `contact` page:
```json
"contact": {
  "elements": [
    {
      "type": "h1",
      "text": "Contact Us"
    },
    {
      "import": "footer"
    }
  ]
}
```
