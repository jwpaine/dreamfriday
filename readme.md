## [dreamfriday.com](https://dreamfriday.com)

A tiny, decentralized, multi-tenant JSON-based CMS for creating and sharing composable UI.

The platform's page rendering engine dynamically constructs a component tree by interpreting JSON data stored in PostgreSQL, keyed by domain. On request, it retrieves the site's complete topology including pages, elements, attributes, styling, and nested structures, then recursively builds and streams the rendered HTML. Styles are aggregated and injected into the document head, with class names generated on the fly to link elements efficiently.

![ALT TEXT](./static/component_chain.png)


### Authentication

Uses the **AT Protocol** for authentications. 
Users may selected between **BlueSky** (default), or supply their own **Personal Data Server** (PDS)

### Routes:

Serialized
- **GET /json**: returns a site's complete structure [Example](https://github.com/jwpaine/dreamfriday.com/blob/main/examples/dreamfriday.com.json)
- **GET /components**: returns all non-private components (PageElements)
- **GET /component/name** retuns a single non-private component (PageElement)
- **GET /page/page_name** returns a page's structure
- **GET /mysites** returns a PageElement containing the list of sites for the logged in user

Rendered:
- **GET /** renders rendered page **'Home'** (ie: domain/pages/Home)
- **GET /page_name** returns rendered page by name
- **GET /login** renders dreamfriday.com/pages/login
- **GET /admin** renders **dreamfriday.com/pages/admin** which imports **dreamfriday.com/mysites**
- **GET /admin/domain** renders site details and JSON editor for specified domain for logged in owner
- **GET /admin/create** renders site creation form

Factory:

- **POST /login** accepts **handle**, **password**, and **server** -> instantiates session and returns cookie
- **POST /admin/create** accepts **domain** and **template** (another domain to cppy).
- **POST /admin/domain"** accepts **previewData** (JSON). Update's preview data for specified **domain**
- **POST /publish/domain** copies **preview** data to **production**
- **GET /logout** destroys current session
- **GET /preview** toggle's preview mode for current session. Page routes will render preview data instead of production

### Topology

## Site

Site data describes the entire site, including all pages and sharable components.

[view site data for dreamfriday.com](https://github.com/jwpaine/dreamfriday.com/blob/main/examples/dreamfriday.com.json)

```JSON
{
  "pages": { "page_name" : Page, "page_name" : Page },
  "components" : { ... }
}
```

## Page

Page structure holds a page's head, body, and a set of redirection flags. 

- **RedirectForLogin** will redirect to the url supplied when logged in. 
  ex: [dreamfriday.com/login](https://dreamfriday.com/login) will redirect to /admin if logged in.
- **RedirectForLogout** will redirect to the url supplied when logged out.
  ex: [dreamfriday.com/admin](https://dreamfriday.com/admin) will redirect to /login if logged out.


```JSON
{
  "head" : { "elements": [ ] }, 
  "body" : { "elements": [ ] }, 
  "RedirectForLogin" : "url", // redirect to url if logged in
  "RedirectForLogout" : "url" // redirect to url if logged out
}
```

## Page Element

```JSON
{
  "type" : "element_type" 
	"attributes" : { "key1" : "value", "key2" : "value2"}
	"elements": [ ]
	"text" : "string"
	"style" :  { "key1" : "value", "key2" : "value2"}
	"import" : "component_name" 
	"private"  false 
}
```

Page elements model any HTML element, including their tag/type (ex: h1, p, a, ..), attributes (ex: id, class), text, styling, and any child elements contained within.

### Imports
Page Elements may import other PageElements. 
**Local imports**: Elements defined within a site's **components** collection may be imported directly by name. 
**remote imports** import can be set to a remotly hosted component (Page Element)

example:
```JSON
{
  "import" : "https://dreamfriday.com/component/Header"
}
```
When a component is imported from a remote source, it will be automatically discoverable via your site's /components route unless **private** is set to true. A good use case for private is if you import data from a protected resource, so that data isn't shared. Example: dreamfriday.com/admin panel imports dreamfriday.com/mysites, which is session-specific. We would not want this data published under dreamfriday.com/components!

Component

```JSON
{
  "component_name" : PageElement,
  "component_name" : PageElement,
  ...
}
```

