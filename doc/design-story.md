This document explains how Lumenta evolved from a photo gallery replacement attempt into a filesystem-first, policy-driven publishing system for a long-term personal image archive.

## Workflow in the Beginning

I am a hobby photographer. I have always been interested in presenting my photos online. That became more important after my children were born: I wanted to show photos of them and my own photos separately, to different audiences.

Some of the images were scanned from film and stored as TIFF files, while the digital compact camera produced JPGs.

### When Everything Still Worked

I discovered Piwigo around the same time as digiKam, and in the beginning they complemented each other quite well. digiKam handled organization, Piwigo handled presentation.

The next step was improving the infrastructure: I got a NAS and a server running in a Docker environment. It was natural that the final storage location for the photos should be the NAS, and that Piwigo should use the original images instead of keeping copies. At the time, this was not directly possible:

- images had to be uploaded
- it could only handle JPG images
- images had to be placed into albums manually

I found workarounds for all three problems:

- after upload, the image paths could be redirected in the database to the mounted NAS directory
- with digiKam batch processing, the TIFF images could be converted to JPG
- I found a plugin that organized images into albums based on tags

At this point I started to realize that I was not using the right tool for the right purpose.
The mismatch was becoming visible, but not yet painful enough to force a rethink.

### Problems Multiply

The number of images grew steadily, and in parallel, the complexity of processing them also increased:

- Rating: especially for children's photos, to decide what was worth showing
- Face marking and tagging, mainly for family members
- An increasingly complex and branching tag hierarchy
- GPS coordinates

And problems also started multiplying on the Piwigo side:

- It does not handle ratings
- GPS handling can be solved with a plugin
- The API did not handle coordinates, so I had to patch the code manually
- Flat tag cloud: I tried to avoid duplicate tag names
- Face marking support can be solved with a plugin

Because tags were assigned to images in waves, meaning new tags were also retroactively assigned to older images, the resynchronization workflow also grew: converting images to JPG, then metadata sync on the Piwigo side, which either succeeded or did not.

In the meantime I reached 30k+ images and 1200+ tags. The system was still manageable: I wrote a few Python scripts that called operations through the API in batch mode, such as metadata refresh or derivative generation.

The deeper problem was that the gallery increasingly became a second organizational system beside the actual archive.

Piwigo 13 came out, and as I discovered, it was able to synchronize from a directory. I started migrating to it.

### The Final Blow

There were several moments where the system finally broke down:

- subdirectory sync worked on an all-or-nothing basis, while my photos of the children were organized chronologically, and the rating should have determined what was included
- because of the derivative system, the original filenames could not be hidden
- the theme I used relied on derivatives that could not be pre-generated, and because of the PHP system, generating derivatives for a page containing many images could stall the entire gallery

And that was when I realized that what I needed was not a web gallery.

The next step was to separate the immediate frustrations from the underlying assumptions that had stopped matching my workflow.

## Reframing the Problem

| Traditional Web Gallery Assumption | Why It Became a Problem for Me |
|---|---|
| Images are uploaded through the web UI, or API | My images already existed in a structured archive |
| Gallery stores its own copies | I wanted the NAS archive to remain authoritative |
| Albums are manually curated | I did not want to organize the same images twice |
| Web UI owns metadata | Metadata already existed in digiKam, EXIF metadata and sidecar files |
| Images are primarily JPG | My archive contained TIFF scans and originals |
| Tags are flat | My archive relied heavily on hierarchical tags |
| Synchronization is import-oriented or slow | Metadata evolved continuously over time |
| Derivatives are generated lazily | Large pages could stall the entire gallery |
| ACLs are managed manually after import | Sensitive images could temporarily remain visible without proper access restrictions |
| The gallery includes many general-purpose social features | Features like registration, comments or original image downloads created unnecessary complexity and additional security concerns |
| Plugins are expected to bridge workflow gaps | The workflow increasingly depended on third-party plugins with different assumptions and maintenance states |

## Ideas Worth Keeping

Not everything about the existing systems was wrong.

Several ideas proved extremely valuable and strongly influenced Lumenta's design.

- A customizable theming system

  I liked the idea that the presentation layer could evolve independently from the backend logic.

- Flexible metadata handling

  As the archive became more complex, configurable metadata and extensibility became increasingly important.

- Clean image-focused presentation

  Even when the architecture no longer matched my workflow, the viewing experience itself often remained simple and pleasant.

## Building a Different System

The new system also had to fit into the infrastructure I was already using.

By this point, the archive already lived on a NAS and most of my services were running in containers.
Docker was not really a development requirement for me - it was an operational requirement.

I wanted the new system to integrate naturally into the existing ecosystem:

- mounted NAS storage
- isolated services
- reproducible deployment
- simple upgrades and maintenance
- straightforward integration into the existing network infrastructure

After all, there is little point in running a web gallery if it is difficult to expose and operate as part of the surrounding system.

---

At this point, the project stopped being "a better gallery idea" and started becoming a real software architecture problem.

Below this line live synchronization pipelines, runtime models, query constraints and operational trade-offs.

The story continues as design.

---

## Designing the System

What started as a collection of small workarounds gradually turned into a fundamentally different set of requirements.

I did not start with a clean architecture. The requirements emerged from the points where the old workflow kept fighting back.

### Emerging Requirements

- Minimal web-side management
- Support for heterogeneous image formats
- Filesystem-first operation
- Deterministic synchronization
- Automatic policy enforcement during synchronization
- Protection against accidental data leakage
- Hierarchical tag support
- Strong security boundaries
- Generated and rule-based structures
- Flexible presentation layer
- Scalability toward large long-term archives

### Minimal Web-Side Management

Anything that could be configured statically should be handled through configuration rather than through the web UI.

The web interface should only manage data that naturally belongs to the runtime state.
Even there, mutable web-side state should remain minimal.

This kept the web application from becoming another large administrative system beside the actual archive workflow.

The benefit was a smaller user interface, less stored mutable state and a system that was easier to configure, move and back up. The cost was that configuration became more important: changes often required restarting the application, configuration validation had to become stricter, and mistakes had to be detected before they affected synchronization or presentation.

### Support for Heterogeneous Image Formats

The archive was never built around a single image format.

It already contained:

- JPG images from digital cameras
- TIFF scans from film
- RAW files from newer cameras

The system therefore could not assume that "web-ready JPG" was the natural form of an image.

### Filesystem-First Operation

The archive already existed on the filesystem long before the gallery.

I did not want the web application to become the owner of the images.
The filesystem remained the authoritative source of the archive:

- originals stayed where they were managed
- files were read in a read-only manner
- metadata was extracted from the files themselves

The gallery increasingly became a projection of the filesystem rather than an independent storage system.

This avoided vendor or application lock-in and prevented split ownership of the same images. The price was that synchronization could no longer be a simple "import everything that exists" operation. It had to support include and exclude configuration, runtime inclusion rules and content-change detection. In this system, that complexity was not an accidental drawback; it was part of the intended behavior.

### Deterministic Synchronization

Once the filesystem became the authoritative source, synchronization stopped being a traditional import process.

Synchronization was no longer about uploading images into the gallery. It had to continuously reconcile the web-visible state with the existing archive.

The same filesystem and metadata state should always produce the same gallery state.

That made traceability part of the design rather than an optional diagnostic feature. If synchronization produced a visible state automatically, the system also needed to explain which inputs, rules and decisions led to that state.

This became one of the central architectural principles of Lumenta: as much of the system as possible should be deterministic and reproducible. The more state could be derived again from the archive, configuration and rules, the less irreplaceable value had to live inside the runtime database. This later shaped the database, upgrade and recovery model.

### Automatic Policy Enforcement During Synchronization

Once synchronization became deterministic and web-side management was intentionally minimized, the synchronization process itself had to become responsible for applying operational rules automatically.

The system needed to derive and enforce properties directly from the archive state, such as:

- access control
- visibility
- inclusion rules
- presentation behavior

This avoided manual post-processing steps after synchronization and kept the resulting gallery state fully reproducible from the archive itself.

### Protection Against Accidental Data Leakage

Once synchronization became responsible for automatically applying operational rules, it also became the natural place to enforce security-related decisions.

The system was designed around the assumption that mistakes and incomplete metadata would eventually happen.
Because of this, fallback behavior had to consistently favor stricter outcomes:

- restricted visibility instead of public visibility
- missing metadata resulting in exclusion rather than inclusion
- absent rules defaulting toward denial instead of exposure

Automation mattered only if it failed safely.

This also connected security back to traceability. Conservative defaults are useful only when their results can be inspected, especially when an image is excluded or restricted because metadata is missing, incomplete or ambiguous.

### Hierarchical Tag Support

Hierarchical tags were already a fundamental part of the archive organization workflow.

The system therefore could not treat hierarchical metadata as an optional extension or as a flattened compatibility layer.
It had to support hierarchical relationships as a first-class concept built directly into the model itself.

It was not enough to display nested tags visually. The system had to preserve and leverage the semantic structure already present in the archive.

### Strong Security Boundaries

The system was designed around the assumption that a self-hosted photo archive may contain sensitive personal images.

Because of this, security-sensitive behavior had to remain explicit and difficult to enable accidentally.

Administrative access was intentionally separated from the gallery itself and designed to integrate with external authentication systems already present in the surrounding infrastructure, such as:

- OpenID providers
- reverse proxy forward-auth systems

Development-oriented behavior also had to remain explicitly enabled rather than existing implicitly in production environments.

This reduced the risk of accidental exposure caused by misconfiguration, incomplete setup or operational mistakes.

### Generated and Rule-Based Structures

Once web-side management was intentionally minimized, manually maintaining the presentation structure no longer made sense either.

The curation process had already happened before the images reached the gallery.

The visible organization of the gallery needed to emerge automatically from the archive itself through explicit rules.

This minimized manual intervention and avoided turning the gallery into another organizational layer beside the archive.

The system increasingly stopped behaving like a manually operated gallery and started behaving more like a rule-driven projection of the archive.

Rules became the mechanism that connected the archive state to the visible gallery state.

The gain was that the whole visible structure could be rebuilt from the archive. The cost was giving up ad hoc manual albums inside the gallery. If a special grouping was needed, the change had to go back to the archive or to the rules that interpret it. If the metadata was wrong, the generated structure would be wrong as well. Rules made organization reproducible, but they also made rule clarity, validation and diagnostics part of the architecture.

### Flexible Presentation Layer

The presentation layer needed to remain adaptable independently from the underlying archive and synchronization logic.

Different visual styles, layouts and presentation approaches should not require changing the core system itself.

Internationalization also needed to be treated as a built-in concern rather than an afterthought.

In practice, that meant separating:

- presentation behavior
- visual appearance
- language-specific rendering

without affecting the underlying archive or synchronization model.

### Scalability Toward Large Long-Term Archives

The system was intended to operate on an archive that would continue growing over many years.

This was not a theoretical requirement. My archive already contained more than 30,000 images, with roughly half of them becoming database-backed publishable images.

This introduced different priorities than those of a typical short-lived or web-oriented gallery:

- long-term maintainability
- predictable operational behavior
- incremental growth
- reproducible synchronization
- minimizing repeated manual work
- query-level performance awareness

The archive needed to remain manageable not only at its current size, but also after years of accumulated images, metadata and organizational complexity.

## System Structure

Before implementation choices, the system needed a vocabulary of boundaries.

The archive itself exists independently from Lumenta.

Images, metadata and organizational structure already exist before the system interacts with them.
Lumenta begins at the synchronization boundary.

### Archive

The external filesystem-based archive:

- images
- metadata
- sidecar files
- directory structure

This is the already established authoritative source described earlier.

### Synchronization & Processing

Synchronization is the first point where Lumenta interacts with the archive.

Its responsibility is to:

- discover changes
- extract metadata
- apply rules and policies
- derive operational state
- maintain consistency

This process transforms the external archive into a reproducible runtime representation.

### Runtime State

The runtime state is the derived operational representation maintained by Lumenta.

It exists to support:

- querying
- rule evaluation
- synchronization tracking
- policy enforcement
- presentation generation

### Presentation

The presentation layer transforms runtime state into user-visible structures and pages.

### Gallery State

The visible web-facing representation derived from the runtime state.

### Configuration

Configuration influences synchronization behavior, metadata handling, rules and operational policies.

### Theme

Themes affect only presentation behavior and visual appearance.

## Defining the Data Model

The first model was deliberately small.

At the beginning, the system only needed to describe the concepts already visible in the gallery domain:

- users
- albums
- images
- tags

### The Initial Gallery Model

The original model was still strongly centered around the gallery itself.

- User

  Defined visibility boundaries and administrative access.
- Image

  The central publishable entity of the system.
- Album

  A presentation-oriented grouping mechanism for images.
- Tag

  Metadata-driven navigation which later evolved toward hierarchical semantics.

### Synchronization Becomes a First-Class Concern

As long as the system behaved mostly like a traditional gallery, the image/album/tag model was sufficient.

But once synchronization became responsible for applying rules, enforcing policies and generating visible state automatically, its behavior also needed to become reviewable and understandable.

Simple logging and console output were no longer enough.
The system needed to preserve:

- what happened
- why it happened
- which rules influenced the result
- and how the final state was produced

Synchronization therefore stopped being merely a background operation and started becoming part of the domain model itself.

- SyncRun

  Represents the state and lifecycle of a synchronization execution.
- SyncFile

  Represents the processing outcome of an individual file during synchronization.

### Remembering What Should Be Ignored

Not every file discovered during synchronization needed to become an image.

At first, excluded files were simply skipped.
But over time this became increasingly wasteful:
the system repeatedly rediscovered and reprocessed files that were already known to be irrelevant.

What started as a simple filtering problem gradually became a performance and operational concern.

The system eventually needed to remember not only what should exist in the gallery, but also what should intentionally remain outside of it.

This led to the introduction of the filtered state.

Filtered state also preserved enough metadata to avoid repeating expensive reads for files whose exclusion decision was already known.

### Final Data Model

The model distinguishes between archival facts, derived presentation state, and operational traceability.

| Concept | Why It Exists |
|---|---|
| User | Access control and administration |
| Image | The publishable representation of an archive item |
| Album | Generated presentation grouping |
| Tag | Metadata-driven navigation and classification |
| SyncRun | Traceability of synchronization executions |
| SyncFile | Per-file sync history and diagnostics |
| Filtered | Remembered ignored inputs to avoid repeated work |

## Implementing the Model

Once the conceptual model started stabilizing, the implementation details also began to emerge more naturally.

The architecture increasingly converged toward:

- a filesystem-backed archive
- a derived runtime model
- a synchronization and processing layer
- and a presentation-oriented web layer

### Runtime State as a Database Model

The runtime state needed to support:

- querying
- synchronization tracking
- rule evaluation
- policy enforcement
- presentation generation

This eventually made a persistent database-backed runtime model unavoidable.

A relational database model fit these requirements naturally, and I eventually chose MySQL/MariaDB.

Beyond the technical aspects themselves, the operational model also aligned well with the surrounding infrastructure:

- containerized deployment was straightforward
- a single database instance could serve multiple independent applications
- schema ownership remained isolated at the database level
- the existing Docker environment already centralized operational database services

### Choosing the Language

As the architecture evolved, the implementation requirements also became increasingly clear.

The system needed:

- a lightweight web layer
- efficient long-running synchronization processes
- strong concurrency primitives
- deterministic processing behavior
- low operational overhead
- simple deployment into containerized environments
- flexible rendering and presentation support

I chose Go mostly because it aligned naturally with these requirements while still remaining relatively small and approachable as a language.

Compared to my previous experience with Java, Go felt:

- significantly simpler
- easier to reason about
- easier to structure incrementally
- much faster to iterate with

Several parts of the ecosystem also matched the emerging architecture particularly well:

- Gin provided a lightweight HTTP and routing foundation
- Go templates aligned naturally with the flexible presentation and theming requirements
- goroutines and channels fit well with synchronization and processing workflows
- static binaries simplified deployment and operational management

I also wanted a project large enough to properly learn and grow into the language.

Other languages I already knew introduced architectural friction in different ways.

PHP remained strongly tied to request-oriented execution models.
The synchronization system behaved more like a continuously operating processing engine than a request/response web feature. Keeping its operational state entirely inside request boundaries, or pushing every intermediate step into storage, would have added unnecessary complexity.

Python made experimentation easy, but the growing amount of interconnected runtime state, synchronization logic and rule evaluation increasingly benefited from compile-time type guarantees and stronger structural constraints.

Java could likely have handled the system technically, but the surrounding ecosystem increasingly pulled toward significantly heavier operational models than what the project required.

### Main Architectural Components

At the highest level, Lumenta split into two large components:

- synchronization: archive -> runtime database
- presentation: runtime database -> web

The archive itself remained outside the system boundary.

The database became the contract, the boundary where synchronization decisions and presentation needs met. The sync process produced the runtime state; the presentation layer revealed which shapes of that state were actually useful for browsing. Later database and query decisions refer back to this boundary.

### Why Synchronization Became a Pipeline

Synchronization could not remain a single import function.

It had to discover files, decide whether they were relevant, extract metadata, apply rules, update runtime state and prepare presentation-related derived data. These steps depended on each other, but they did not all have the same responsibility.

The stages also had different resource and time requirements. Filesystem discovery was mostly I/O-bound, hash calculation could be CPU- and I/O-heavy, metadata extraction could be expensive, and database operations had their own latency and contention profile.

The pipeline made those cost profiles explicit. Each stage could later be backed by its own workers and limits, while the ordering between stages still preserved the decisions required by the synchronization model.

That made the synchronization process a natural pipeline.

### Synchronization Pipeline

At a high level, each discovered file moves through the following stages:

1. Discovery: find the file and create the work item.
2. Hash Calculation: calculate the hashes needed for change detection.
3. Database Lookup: load the previous runtime state.
4. Dirty Check: decide whether the file needs further work.
5. Metadata Read: read metadata only when necessary.
6. Runtime Inclusion Decision: decide whether the file enters the image runtime state.
7. Property Derivation: derive additional image properties.
8. ACL Derivation: derive access-control state.
9. Image State Write: write the image runtime state.
10. Post-Processing: update secondary generated structures.
11. Trace Recording: record both item-level and process-level outcomes.

#### 1. Discovery

Finds files in the filesystem and creates a work item for each discovered file.
Discovery starts from configured roots: named filesystem entry points where synchronization is allowed to look for archive files.
This stage also reads filesystem properties such as path, size and modification time, because these properties are part of the identity and change detection input.

#### 2. Hash Calculation

Calculates the content hash or metadata hash needed to compare the discovered file with previously known state.

#### 3. Database Lookup

Loads the existing runtime state for the file, if any.
This may include an existing image record, previous sync information or a remembered filtered record.

#### 4. Dirty Check

Compares filesystem properties, hashes and existing database state.
This stage decides whether the file is new, unchanged, modified, already filtered or needs further processing.

The decision is not based only on the file's internal state. Some synchronization decisions depend on rules loaded from configuration, so a relevant configuration change can force reevaluation even when the file itself has not changed.

#### 5. Metadata Read

Reads image metadata only when the dirty check indicates that it is necessary.
This avoids expensive metadata work for files whose existing state is still valid.

This is also why filtered files became part of the runtime model instead of being treated as temporary skips. When a file is not dirty, the pipeline can reuse metadata already stored in either the image state or the filtered state instead of reading the original file again.

#### 6. Runtime Inclusion Decision

Applies the first rule-based decision.
At this point the system decides whether the file should become or remain part of the image runtime state, or whether it should be remembered as filtered.
Depending on the status, the decision may use filesystem data, freshly read metadata, or existing database state.

Because the inclusion rules come from configuration, previously filtered items are not always final. If the configuration that controls inclusion changes, this stage must reevaluate the item independently from its cached filtered state.

This is the first branching point of the pipeline. Files rejected by this decision do not continue through property derivation, ACL derivation, image writes or album assignment. Instead, they enter a shorter filtered-item path that updates filtered state and records the exclusion decision.

#### 7. Property Derivation

Derives additional image properties from the available data.
Currently this includes properties such as the panorama flag.

#### 8. ACL Derivation

Applies rule-based access control decisions.
Visibility is derived before the image is written, so the stored runtime state already reflects the policy outcome.

#### 9. Image State Write

Creates or updates the image record in the database.

#### 10. Post-Processing

Applies secondary derived structures, such as album assignment.
These operations are also rule-based and depend on the image state already being known.

#### 11. Trace Recording

Records the outcome of the processed item as a sync file entry and closes the overall sync run when the process finishes.

Traceability applies to both accepted and rejected inputs. The system should explain not only why an image exists in the runtime database, but also why a discovered file was intentionally kept out of it.

### Presentation Layer

The presentation layer was not just a consumer of the data model. It became one of the forces that shaped it.

Once the archive had been translated into runtime state, the next question was how that state should be exposed as a navigable web application. This changed the role of the database: it was no longer only a synchronization target, but also the read model of the presentation layer.

The presentation layer reads from the database, not from the archival filesystem. It does not own the originals and it does not modify them. Its responsibility is to turn synchronized runtime state into public and administrative views.

That boundary kept domain decisions out of templates. Templates could focus on rendering view data, while routes and data-loading code selected the correct runtime state for the current user and page.

This made several presentation concerns part of the data model discussion:

1. Album navigation needed a structure that could support trees, descendants and breadcrumbs.
2. Image pages needed stable previous and next navigation inside albums and tags.
3. Tag pages needed their own image lists and aggregation logic.
4. Public views had to be access-control aware.
5. Paged lists needed matching count queries and deterministic ordering.
6. Web delivery needed derivatives and thumbnails instead of direct use of original files.

Presentation therefore became a design input, not just the final skin over the data.

### Presentation Technology

The next major decision was to keep presentation mostly server-rendered.

Lumenta did not need a client-side application framework for its core browsing experience. The main interaction model was navigation through albums, images, tags and administrative pages, all of which mapped naturally to HTML pages generated from database-backed view data.

This led to a deliberately simple presentation stack:

1. Go templates render the primary pages.
2. HTML remains the main interface contract.
3. JavaScript is used in isolated places where local interaction needs it.
4. REST endpoints exist mainly to support those JavaScript islands.

Even small presentation details followed this pattern: icons are centrally defined, rendered through templates and exposed through a helper function.

The goal was not to avoid JavaScript entirely, but to keep it from becoming the application boundary. The server owns routing, page composition, permissions and the main view model; JavaScript only enhances specific interactions where that trade-off is worth it.

The decision was also based on earlier experience with a fully separated HTML and JavaScript application backed only by a REST API. That approach worked for a simpler case, but it did not have to carry a user and security model, translations or replaceable themes. Adding those concerns would have moved a large amount of application structure into the API and client-side state management. Lumenta still has an API layer where it is useful, but making the API the primary boundary would have added complexity before the project needed it.

The visual model also pushed toward server-rendered structure with CSS-driven behavior. A central part of the browsing experience is a weighted, ordered masonry-like grid. The layout has to remain responsive across different resolutions and viewing contexts, and the most direct way to express those constraints is through HTML, CSS and injected CSS variables. Moving that responsibility primarily into JavaScript would have turned layout into another application subsystem.

Finally, the application was intentionally designed to minimize administrative and configuration UI. Most pages are read-only views over synchronized state. At this point, only a small minority of endpoints mutate data, while the rest serve navigation and presentation. For that shape of application, a modern client-side framework would have introduced a second application model without carrying enough of the system's core responsibility.

### Navigation-Driven Queries

The shape of the user interface turned directly into database access patterns.

Browsing requirements became query requirements: breadcrumbs needed ancestry, paged grids needed counts, image pages needed stable previous and next navigation, and every public list needed the same access-control constraints.

This made the presentation layer an important source of pressure on the runtime model. A purely conceptual image, album and tag model was not enough; the database also had to support the way users actually move through the gallery.

The result was a read model shaped around navigation:

1. Album pages need descendants, image counts and breadcrumb data.
2. Tag pages need image lists and aggregation data.
3. Image pages need context-sensitive previous and next items.
4. Public pages need the same ACL rules applied consistently.
5. Grid pages need deterministic ordering and matching count queries.

At that point, database design stopped being only about storing entities. It became about supporting the access patterns that synchronization produced and presentation required.

### HTML-First, Not Static

HTML-first did not mean static pages.

It meant that routing, permissions, language selection, theming and page composition remained server-side. JavaScript was reserved for local interaction where it improved the page without taking ownership of the application model.

This kept the main user experience simple to reason about. A page request produced a page from a known runtime state, under known permissions, in a known language and theme. The dynamic parts of the interface could then be added as local enhancements rather than as a separate client-side application.

## Security Model

Security in Lumenta is built around explicit ownership and conservative access rules.

The administrative surface is not meant to stand on its own as a public authentication system. Admin access requires external authentication infrastructure. Without that boundary in front of it, the admin area should not be exposed.

Admin users are also not created through the web interface. They are bootstrapped through the command line, which keeps administrative identity creation outside the publicly reachable application flow.

An administrator is treated as the owner of the archive and the system. This is not a delegated permission model where an admin merely receives access from another user. By definition, the administrator can access every image, because the administrator is responsible for the archive, synchronization rules and resulting presentation state.

### Access Levels

The access-control model has four levels:

1. public: visible to everyone
2. logged-in: visible to authenticated users
3. user-specific: visible to a dedicated user
4. admin: visible to administrators

This keeps the model small enough to reason about, while still covering the main privacy boundaries needed by a personal archive.

### Album ACL and Image ACL

Albums and images have separate access-control rules.

Album ACL controls the visibility of the album as a presentation structure. It decides whether the album itself can be seen as a navigational object.

It does not automatically grant access to child albums.
It also does not grant access to the images assigned to the album.

Image visibility is always evaluated from the image's own ACL rules.

This separation is important because albums are presentation mappings, while images are the actual publishable objects. A user may be allowed to see a structural part of the gallery without automatically receiving access to every image connected to it. Conversely, image access must remain correct no matter which album, tag or page leads to the image.

## Database Design

The database was designed around a strict distinction between valuable state and reproducible state.

Most database content is not treated as irreplaceable. Images, tags, filtered records, synchronization traces and derived presentation data can be produced again from the archive, configuration and rules. They matter operationally, but they are not the source of truth.

The main exceptions are users and albums.

Users carry access and administration identity inside Lumenta. Even when authentication is delegated to surrounding infrastructure, the local user model still defines how external identity maps to application-level access.

Albums carry manually defined presentation intent:

1. names
2. hierarchy
3. access-control settings
4. rules that connect archive state to presentation structure

That makes the `users` and `albums` tables the parts of the database that currently contain valuable non-reconstructable state. Everything else is either derived from the archive or exists to make synchronization and presentation queryable.

This also shaped the upgrade strategy. Because the system is deterministic and the runtime state is reproducible, database upgrades do not have to start from a heavy migration path for every derived table.

The expected recovery or upgrade path is deliberately simple:

1. export the user and album state
2. create a fresh database
3. import the users and albums
4. run synchronization
5. let the runtime state be rebuilt

This does not remove the need for schema care, but it changes where the risk sits. The important part is preserving the small amount of intentional state; the rest of the database can be regenerated when the archive, configuration and rules are still available.

After the user and album state has been created and exported, a full database backup becomes much less valuable than it would be in a traditional application. It can still be useful as an operational shortcut, but it is no longer the primary recovery mechanism. The durable state lives in the archive, the configuration, the rules and the saved user and album definitions; the database is mostly a rebuildable runtime projection.

### Identity and ID Strategy

The database still uses internal record IDs.

Those IDs are useful inside the runtime model: they make joins simple, keep relationships compact and give DAO code stable references to database records. In that sense, an `image_id`, `album_id`, `tag_id` or `user_id` is an internal implementation identity.

But for archive-derived data, the database ID is not the identity that matters outside the system.

From the outside, an archive item is identified by its filesystem position and file identity:

1. root
2. path
3. filename
4. extension

Here, `root` means a named filesystem entry point from which synchronization discovers archive files.

Together these describe where the file belongs in the authoritative archive. They are the identity used when synchronization discovers, compares and reconstructs runtime state.

This distinction matters because the database is rebuildable. A fresh database may assign different internal record IDs after synchronization, but the same archive file should still be recognized as the same input when its external identity is unchanged.

Internal IDs therefore belong to the runtime database model. Archive identity belongs to the synchronization boundary.

### Identity vs Derived Identity

Not every identifier in the system has the same meaning.

Some identities are intentional. Users and albums are created to express application-level intent: who can access the system, and how the archive should be presented. Their identity has to survive database rebuilds, because it represents decisions that cannot be reproduced from the archive alone.

Other identities are derived. Images, tags, filtered records and generated relationships are reconstructed from archive files, metadata, configuration and rules. Their database records need stable IDs while the runtime database exists, but those IDs are not the durable identity of the underlying concept.

For images and filtered records, the durable identity comes from the archive side: root, path, filename and extension, combined with synchronization state such as hashes and timestamps when change detection is needed.

For tags, identity comes from metadata. A tag exists because the archive metadata describes it, not because the web application created it manually.

For generated relationships, such as album-image and image-tag bindings, identity is even more clearly derived. They exist because the current rules and metadata produce them. If the rules or metadata change, the relationship can disappear and later be created again.

Synchronization trace records have a different kind of identity. They describe events and decisions, not durable domain objects. Their value is diagnostic: they explain how a particular synchronization run produced its outcome.

This separation kept the rebuild model coherent. Rebuilding the database may change internal record IDs, but it should not change the meaningful identity of users, albums or archive-derived inputs.

### Database Tables as Architectural Roles

The schema is easier to understand by responsibility than by table order.

At a high level, the tables fall into five groups:

1. Intentional state: `users`, `albums`
2. Derived publishable state: `images`, `tags`
3. Generated relationships: `album_images`, `image_tags`
4. Remembered exclusions: `filtered`
5. Synchronization traceability: `sync_runs`, `sync_files`

This grouping reflects the system boundary. Some tables preserve intent, while others make the synchronized archive queryable, navigable or explainable.

#### users

`users` stores the local application identity used by Lumenta.

Authentication may be handled by external infrastructure, but the application still needs a local representation for access control, administration and visibility decisions.

#### albums

`albums` stores manually defined presentation structure.

Albums are not merely folders. They carry hierarchy, names, access-control settings and rules that decide how synchronized images become part of the visible gallery structure.

#### images

`images` stores the publishable runtime representation of archive files.

Image records are produced by synchronization. They contain the state needed by presentation, access control, navigation and derivative generation, but they can be rebuilt from the archive when the rules and configuration are still available.

#### filtered

`filtered` stores discovered files that intentionally do not become images.

This table avoids repeatedly reprocessing known exclusions and preserves enough state to explain why a file was kept out of the gallery. It also lets synchronization reuse previously read metadata when the file has not changed.

#### tags

`tags` stores metadata-derived classification and navigation state.

Tags originate from the archive metadata rather than from web-side curation. They are part of the generated browsing model, especially where hierarchical metadata has to remain meaningful.

#### album_images

`album_images` stores the generated relationship between albums and images.

This relationship is derived from album rules and synchronized image state. It supports album pages, image counts, album-scoped navigation and presentation queries.

#### image_tags

`image_tags` stores the generated relationship between images and tags.

It supports tag pages, tag-scoped image lists and tag-based navigation without making the web application the owner of the tagging workflow.

#### sync_runs

`sync_runs` stores process-level synchronization history.

It records the lifecycle and outcome of synchronization executions, making synchronization reviewable as an operational process rather than an invisible background task.

#### sync_files

`sync_files` stores item-level synchronization outcomes.

It records what happened to individual discovered files and why, including accepted images and rejected or filtered inputs.

The schema therefore mixes intentional state, derived state and traceability state. That was intentional: the database is not only a persistence layer, but the operational contract between synchronization and presentation.

### Column Shape Decisions

After the table responsibilities were clear, the next question was how structured the data inside each table should be.

Dedicated columns and JSON fields optimize for different things.

Dedicated columns keep data easy to query, filter, order and update. They are the natural choice for common facts, presentation access patterns, synchronization control fields and values that participate directly in ACL or navigation logic.

JSON fields preserve more complex structures without forcing every internal shape into the relational model. They are useful for rarely queried data, rule definitions, diagnostic records and cached structures whose internal format may evolve. The cost is also clear: simple SQL queries become harder, updates are less direct, and the application gains an extra serialization and deserialization layer.

The practical rule was to use columns for frequently used facts and JSON for structured data that is either complex, rarely queried or mostly consumed by a specific part of the application.

#### images

The `images` table keeps common image facts and photo metadata in dedicated fields when they are useful for synchronization, presentation or querying.

The full metadata set remains JSON. It can be large, source-dependent and more detailed than the presentation layer usually needs. Keeping it as structured metadata preserves the information without turning every possible metadata field into schema.

#### albums

The `albums` table keeps most of its state in dedicated fields, because album data is valuable, intentional and directly affects navigation, access control and presentation.

The main exceptions are ancestor information and rules.

Ancestor data is stored as a structured list to support descendant and breadcrumb handling without repeatedly walking the hierarchy. Rules are also structured data: they are complex, rarely queried directly and primarily interpreted by synchronization logic.

#### tags

The `tags` table does not need JSON.

Tag state is part of the generated navigation model and is simple enough to fit naturally into dedicated fields.

#### filtered

The `filtered` table stores metadata as JSON for the same reason `images` stores full metadata as JSON.

Filtered records are not publishable images, but they still need to remember enough metadata to avoid expensive rereads and to explain exclusion decisions. The metadata cache has the same role as image metadata: it preserves source-derived structure without making every field queryable.

#### sync_runs

The `sync_runs` table does not need JSON.

It represents process-level synchronization state, where the important facts are lifecycle, timing and outcome fields.

#### sync_files

The `sync_files` table stores detailed decision traces as structured data.

Those traces can be complex and are usually read for diagnostics or administrative inspection, not for ordinary presentation queries. When they are needed, there is time to deserialize and present them in an admin view. Keeping them structured avoids forcing every intermediate synchronization decision into first-class columns.

### Query Model and DAO Shape

The DAO layer was shaped around explicit database access paths.

This mattered because the system was designed for a large archive, not only for a small personal gallery. With tens of thousands of source images and many thousands of publishable records, query-level optimization became part of the architecture.

The database is the contract between synchronization and presentation, so most queries exist because a specific workflow needs them. Synchronization needs lookup, comparison, insertion and trace updates. Presentation needs album pages, tag pages, image detail pages, paging, counts and navigation.

This made the query model intentionally explicit. The system should know where a query starts, which tables it needs and which constraints belong in the database.

Public presentation queries are access-control aware. They should only return records visible to the current viewer instead of loading a broader set and filtering later in templates.

Paged views usually need matching query pairs: one query to load the current page and another to count the matching records. This keeps pagination tied to the same constraints as the visible list.

Image pages need context-sensitive previous and next queries. The previous or next image depends on whether the viewer arrived through an album, a tag or another navigational context.

Album navigation needs ancestor and descendant queries. Breadcrumbs, subtree counts and album-scoped image lists all depend on the database being able to answer tree-shaped questions efficiently enough for presentation.

For example, loading images for an album should be able to start from the album-image relationship using the known `album_id`. It should not have to touch the `albums` table if the album row itself is not needed for that operation.

The result is a DAO layer with many purpose-built queries. That is deliberate. The goal is to make the important synchronization and presentation access paths explicit, reviewable, optimizable and testable.

## Operational Model

Lumenta was designed to fit into the environment where the archive already lived.

The operational model is built around a few stable assumptions:

1. the archive lives on mounted filesystem storage
2. the application runs as an isolated service
3. the database stores runtime and intentional state
4. synchronization can rebuild most runtime data
5. presentation reads from the rebuilt runtime model

This keeps the system close to the infrastructure that already existed: NAS storage, containerized services, reverse proxy routing and shared operational database services.

### Normal Operation

During normal operation, the archive remains outside Lumenta.

The web application serves presentation views from the database. It does not scan the filesystem during page rendering, and it does not modify original files.

Synchronization is the process that connects the two worlds. It reads the archive, evaluates configuration and rules, updates runtime state and records what happened.

This creates a clear operational split:

1. archive changes happen outside the web application
2. synchronization turns those changes into runtime state
3. presentation serves the current runtime state

### Distributed Operation

The same split also makes distributed operation possible.

The web server does not need to perform the most expensive archive processing during normal page requests. Once derivatives and runtime state exist, serving a page is mostly database queries, permission checks and template rendering. That means the public-facing Lumenta web service can run in a relatively small container.

Synchronization has a very different resource profile. It needs filesystem I/O, hashing, metadata extraction, rule evaluation and derivative-related work. Those tasks can benefit from running close to the archive and on a machine with more available CPU, memory and disk throughput.

Because synchronization is a separate operational concern, another binary can drive the sync process from a more suitable machine, such as a workstation used for photo processing. That machine can do the heavy archive scan and update the shared runtime database, while the web container remains focused on serving the current state.

This keeps the web-facing service small without forcing the synchronization pipeline to run under the same resource constraints.

### Rebuild and Recovery

Because most runtime state is reproducible, recovery does not depend on preserving every database row.

The durable operational inputs are:

1. the archive
2. configuration
3. rules
4. exported users
5. exported albums

With those available, the rest of the runtime database can be rebuilt by running synchronization again.

This does not make database backups useless. A backup can still be the fastest way to restore service. But it changes the primary recovery model: the system is designed so that losing derived runtime state is inconvenient, not catastrophic.

### Upgrade Strategy

The same idea shapes upgrades.

Instead of treating every database table as long-lived irreplaceable state, the system can tolerate replacing derived tables with a fresh schema and rebuilding them from the archive.

The parts that need more careful handling are the intentional parts: users and albums. Those express access and presentation intent, so they must survive upgrades through export, import or explicit migration.

This makes schema evolution less frightening. The database still matters, but its most valuable state is intentionally small.

### Administrative Role

The admin interface is not primarily a content-management surface.

Most content and presentation state comes from synchronization, not manual editing. The administrator's operational role is therefore closer to system owner and reviewer:

1. maintain users and album intent
2. inspect synchronization results
3. review filtered or rejected inputs
4. verify access-control outcomes
5. trigger or supervise rebuild-oriented workflows

This follows from the earlier decision to keep web-side management minimal. Administration exists, but it is mostly about ownership, diagnostics and correction of the model inputs rather than direct editing of every visible object.

## Diagnostics and Reviewability

Deterministic synchronization is only useful if its decisions can be reviewed.

Lumenta therefore needs two complementary diagnostic layers:

1. persistent synchronization trace, stored in the database
2. runtime logging, used to understand execution flow while the system is running

The persistent trace explains domain-level outcomes. It records synchronization runs, individual file decisions, accepted images, rejected inputs and filtered files. This makes it possible to inspect why a file became part of the gallery, or why it was intentionally kept out.

Runtime logging answers a different question: what happened while the code was executing.

This became important because synchronization is strongly parallelized. Multiple files can move through different stages at the same time, and a plain stream of log lines quickly becomes difficult to read. Without correlation, the output becomes an unordered pile of events.

That pressure led to a structured, process-oriented logging model built on top of `zerolog`. Related log events carry enough identifiers to reconstruct the call tree even when goroutines and background workers interleave their output.

The amount of output also became a problem on its own. A single global log level was too blunt, so logging needed hierarchical scope-based level control. Flood control also needed a human-readable anchor, such as a filename, to keep the operational outline readable when lower-level details were suppressed.

The resulting package stayed lighter than a full tracing or APM system.

Over time, this stopped being only a Lumenta helper. The same need for scoped, contextual and flood-resistant logging applied outside the gallery as well. The logging code eventually grew into a standalone Go package.

## What the Architecture Optimizes For

Lumenta is not a general-purpose gallery platform.

Its architecture is optimized for a specific kind of problem: publishing a large, long-lived personal archive without making the web application the owner of that archive.

### It Optimizes For

Long-term archive ownership.

The original files remain outside the application and stay under the archive workflow that already existed before Lumenta.

Deterministic rebuilds.

Most runtime state can be produced again from the archive, configuration, rules and the small amount of intentional state stored in users and albums.

Privacy and conservative defaults.

Missing or incomplete metadata should not accidentally expose images. Access-control decisions are derived and enforced as part of synchronization and query behavior.

Low web-side management.

The web interface is not meant to become a second photo management system. Most visible structure is generated, and administration focuses on ownership, review and diagnostics.

Query-aware presentation.

The presentation layer is designed around real browsing needs: albums, tags, breadcrumbs, paging, counts, previous and next navigation, thumbnails and derivatives.

Operational flexibility.

The web server can stay relatively small, while synchronization can run where CPU, memory and filesystem throughput are more appropriate.

Reviewability.

Synchronization should not be a black box. The system records enough state to explain both accepted and rejected inputs.

### It Does Not Optimize For

General social-gallery features.

Registration, comments, public uploads and social workflows are intentionally outside the core problem.

Arbitrary web-side curation.

The system does not try to make every visible structure editable from the browser. If the archive or rules are the source of the structure, changes should happen there.

Rich client-side application behavior.

JavaScript is used where local interaction needs it, but the application boundary remains server-side.

Perfect preservation of derived database rows.

The database matters, but most of it is a rebuildable runtime projection. The system protects intentional state more strongly than derived state.

Universal multi-tenant flexibility.

The security model is designed around a personal archive with explicit ownership, not around a generic SaaS permission model.

The result is a system with a narrow but deliberate shape. It trades generality for ownership, reproducibility, privacy and operational clarity.

## Current State and Open Decisions

Lumenta is still evolving.

The core architectural direction is already visible: filesystem-first archive ownership, deterministic synchronization, a rebuildable runtime database, server-rendered presentation, explicit security boundaries and reviewable processing.

Several parts are already strong enough to validate the design:

1. the archive can remain outside the application
2. synchronization can derive runtime state from external files and metadata
3. filtered inputs can be remembered instead of repeatedly rediscovered
4. presentation can be driven from the runtime database
5. users and albums can remain the small set of intentional database state
6. diagnostics can explain both accepted and rejected synchronization outcomes
7. the sync files view can inspect per-file state and individual rule evaluations
8. rebuild performance is already practical, with synchronization reaching around 100 MB/s even over NAS storage, and higher throughput when running from SSD-backed local storage in my current setup.

At the same time, the project is not finished.

The most important open areas are:

1. export and import tooling for users and albums
2. refining the responsive visual layout and masonry grid behavior
3. expanding test coverage around templates, DAO queries and synchronization decisions
4. continuing to document configuration, rules and operational procedures

Some decisions are intentionally still allowed to evolve.

The presentation layer can keep changing as long as it stays behind the same runtime model boundary. The database schema can still be adjusted where the derived read model needs better access paths.

The important part is that those changes now have a direction.

The project no longer depends on finding the right gallery plugin or bending an existing system into shape. It has its own architectural center: an archive remains the source, synchronization derives a reviewable runtime model, and presentation publishes that model without taking ownership of the archive itself.
