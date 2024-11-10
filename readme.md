# Dream Friday

**Dream Friday** is a simple, JSON-based CMS (Content Management System) built with Go and PostgreSQL. It enables dynamic page rendering through JSON configuration, allowing for reusable components, flexible styling, and responsive design.

## Features

- **JSON-based Configuration**: Define pages, components, and styling directly in JSON, making updates easy and code-free.
- **Reusable Components**: Create and reuse components across pages.
- **Dynamic Styling and Media Queries**: Use inline styles and responsive media queries to customize each element.
- **Dynamic Pages**: Each page is rendered based on JSON data managed by the CMS and stored in PostgreSQL.
- **Preview Mode**: Easily test unpublished changes before going live by enabling preview mode.
- **Built with Go, PostgreSQL, and Auth0**: Ensures performance, scalability, and secure login.

## How It Works

Dream Friday uses JSON to define page structures and reusable components, stored in PostgreSQL and managed through an admin interface. Key features include:

- **Components**: Reusable elements (like buttons or headers) that can be imported into multiple pages.
- **Pages**: Define each page’s layout and content.
- **Attributes and Styling**: Apply styles and attributes directly to each element.
- **Imports and Overrides**: Reuse components within pages and override specific properties.
- **Media Queries**: Define responsive styling directly in JSON.
- **Preview Mode**: Test unpublished content updates at `/preview`.

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

#### How to Use Preview Mode

1. **Login**: Use Auth0 for secure authentication. Only logged-in users with admin permissions can access preview mode and edit content.

2. **Enable Preview**: Go to `/preview` to enable preview mode. This will fetch any unsaved drafts from the database.

3. **Edit and Save**: Use the JSON editor in the admin page to make changes and save them to preview mode. This allows you to review the updates without affecting the live site.

4. **Publish Changes**: When ready, publish the changes from the admin page to make them live.

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