# Dream Friday

**Dream Friday** 

A decentralized, multi-tenant, JSON-based, CMS for creating and sharing composible UI. 

Page endering engine dynamically constructs a component tree by interpreting JSON data stored in PostgreSQL keyed by domain. Upon request, data defining a site's complete topology, including pages, page elements, including their attributes, styling, and children, and recursively builds and streams rendered HTML. Styling is aggregated and injected into the document head, linking elements by class names generated on the fly.

### Authentication

Uses the **AT Protocol** for authentications. 
Users may selected between **BlueSky** (default), or supply their own **Personal Data Server** (PDS)

### Routes:

Serialized
- **GET /json**: returns a site's complete structure
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

Site

```JSON
{
  "pages": { "page_name" : Page, "page_name" : Page },
  "components" : { ... }
}
```

Page

```JSON
{
  "head" : { "elements": [ PageElement, PageElement, ...] }, 
  "body" : { "elements": [ PageElement, PageElement, ...] }, 
  "RedirectForLogin" : "string", 
  "RedirectForLogout" : "string" 
}
```

Page Element

```JSON
{
  "type" : "element_type" 
	"attributes" : { "key1" : "value", "key2" : "value2", ... }
	"elements": [ PageElement, PageElement, ...]
	"text" : "string"
	"style" :  { "key1" : "value", "key2" : "value2", ... }
	"import" : "component" // may be locally defined, or set to somedomain.com/component/name
	"private"  bool // if set to true, the import referenced will not be made available for public export via /components/name
}
```
Component

```JSON
{
  "component_name" : PageElement,
  "component_name" : PageElement,
  ...
}
```

