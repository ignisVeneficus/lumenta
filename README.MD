# Lumenta

**Lumenta is a rule-driven, self-hosted gallery engine for people who already manage their photos properly — and don’t want to manage them again in their gallery.**

It is not just a gallery.  
It is a structured rule system from which albums are generated automatically.

You tag once.  
You organize once.  
You define rules once.

After that, you just add images.

No manual syncing.  
No album micromanagement.  
No per-image ACL tweaking.

Lumenta reads your filesystem directly, extracts metadata, stores it in a database, and evaluates your rules to build albums automatically.  
The web layer operates exclusively from this indexed model — reflecting the structure you already created.

Designed for large photo collections, Lumenta removes repeated work and long-term maintenance overhead.

> Configure it once. Feed it images. Forget it.

Automation. Without chaos.

## Where Lumenta fits in a photographer’s workflow

Lumenta typically enters at the end of your workflow — **after editing and tagging**.

A typical process:

1. Shoot
2. Import into your photo management tool (e.g., Lightroom, digiKam, etc.)
3. Select and edit
4. Rate, tag and organize
5. Export or sync to your structured archive
6. Lumenta reads the archive automatically

There is:

- No manual upload step  
- No web-format conversion step  
- No manual import  
- No manual album maintenance  

You simply point Lumenta to your existing filesystem-based archive, and it builds the web presentation layer on top of it.

## Why Lumenta is not “just another photo album”

Lumenta follows a **single-owner model**.

This means:

- It is not a multi-user upload platform  
- There is no collaborative image uploading  
- There is no group editing  
- Users exist only with different read-level permissions (e.g., guest, authenticated user)

The archive belongs to one photographer.  
Access levels control visibility — not authorship.

## Filesystem-first design

The filesystem is the source of truth.

- Images remain where you manage them  
- No re-upload required  
- No proprietary storage layer  
- Your directory structure retains its meaning  

Lumenta scans the configured root directory and builds its internal model from that structure.

## Rule-based automation

Rules and automation are central.

Rules define:

- Which images are indexed (and which are excluded)
- What qualifies as special formats (e.g., panorama detection)
- Default ACL (Access Control List) behavior based on metadata or structure
- Album population logic

Albums are not curated manually.  
They are generated deterministically from rule definitions.

## Metadata and hierarchical tags

Lumenta provides a flexible and configurable metadata system.

- Metadata definitions are configurable
- Most rule definitions are built on top of metadata
- Hierarchical tags are fully supported

Hierarchical tagging — still uncommon in many web galleries — is treated as a first-class concept.

The goal is not to re-edit metadata in the web layer, but to faithfully represent and leverage the structure already created in your photo workflow.

Both embedded metadata and sidecar files are supported, with a clear hierarchy between sources when resolving conflicts.

## Strong configuration model

The entire system is built on a powerful configuration layer.

- Indexing behavior
- Rule logic
- Metadata mapping
- Album generation
- Derivative handling
- Default access rules
- Branding
- Theme support

[Configuration documentation](./doc/config.md)

## Presentation layer and extensibility

The web layer is template-based.

- Pages are rendered using configurable templates
- The structure allows future theme support
- The goal is visual flexibility without compromising backend determinism

## Development status

This project is in **early development**.  
The focus so far has been on validating the core architecture, data flow, and abstractions rather than on feature completeness.

The following areas are implemented or actively being refined and are usable, though still evolving:

### Data ingestion and backend core

The complete filesystem → database ingestion path is implemented and under active refinement:

- Database schema and access layer, focused on the `images` domain model
- Filesystem scanning and synchronization pipeline
- Deterministic import driven by a rule engine
- Structured configuration loading and validation

Together, these form the backbone of the system: the authoritative **filesystem → database** data flow.

### Web backend and presentation layer

The server-side web foundation is functional:

- Web framework and routing
- User handling and access control foundations
- Image grid rendering and basic gallery presentation

These components provide the foundation for browsing and interacting with the indexed collection, while higher-level UI features and refinements will evolve over time.

## Showcase

### Public hierarchical tag tree
![public tag tree](./doc/img/tags.png)

### Public hierarchical tag browser
![public tag view page](./doc/img/tag-view.png)

### Admin filesystem browser
![admin file browser](./doc/img/admin-fs.png)

### Admin image details page
![admin image page](./doc/img/admin-img.png)


## Build

```bash
go build ./...
```

This repository should be considered experimental.