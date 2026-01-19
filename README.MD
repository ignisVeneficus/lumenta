# Lumenta

Lumenta is a self-hosted photo library and gallery backend written in Go.  
Its primary goal is to provide a clean, automation-first alternative to traditional photo gallery systems.

The project focuses on deterministic data ingestion, rule-based album generation, and long-term maintainability for large photo collections.

There is currently no test phase or stable release. The project is in an early, architecture-focused development stage.

---

## Core principles

### No manual album curation

Albums are not managed by hand.  
All album–image relationships are generated exclusively by a rule engine and stored as materialized relations.

If an image appears in an album, it is because it matches the album’s rules.

---

### Rule-based albums

Albums are defined using logical rule trees that can combine conditions such as:

- tags (any / all / none)
- time-based conditions (year, date ranges)
- explicit exclusions
- exclusion of images that appear in child albums

Rules are evaluated deterministically and can be expressed in YAML or JSON.

---

### Bottom-up evaluation

Album evaluation is performed from leaf albums toward the root.  
This enables correct handling of exclusions and parent–child relationships without ambiguity.

---

### Read-only tag model

Tags are imported from external tools (for example, digiKam) and treated as a controlled vocabulary.

- Tags are hierarchical and represented by their path structure.
- Tags are not edited inside Lumenta.
- The only supported tag mutation is cleanup of orphaned tags (tags without images and without children).

---

### Filesystem as source of truth

The filesystem is the authoritative source for media files.

A sync pipeline walks the configured root directory and reconciles filesystem state with the database using fingerprints and hashes.

---

### Lazy derivatives

Image derivatives (thumbnails, previews, web sizes) are generated on demand and cached on the filesystem.

No derivatives are generated eagerly unless explicitly requested.

---

## High-level architecture

### Components

- **Filesystem walker**  
  Discovers files under the configured root and produces work items.

- **Sync pipeline**  
  A worker-based pipeline responsible for:
  - path discovery
  - database lookup
  - change detection
  - hashing
  - metadata extraction
  - database upserts
  - relationship updates

- **Rule engine**  
  Evaluates album definitions and produces materialized album–image mappings.

- **Database (MariaDB / MySQL-compatible)**  
  Stores canonical image records, metadata, tags, albums, and materialized relations.

---

## Data model overview

Key concepts in the database design:

- `images`  
  Canonical representation of a media file.

- `tags`  
  Hierarchical tags with parent references and uniqueness enforced per parent.

- `image_tags`  
  Many-to-many mapping between images and tags.

- `albums`  
  Album tree structure.

- `album_images`  
  Materialized output of the rule engine.

The schema is expected to evolve as the project matures.

---

## Configuration

Configuration is intended to be loaded from YAML (and optionally JSON) into a single configuration structure.

Typical configuration areas include:

- server settings (address, timeouts)
- database connection and pooling
- media root paths
- album rule definitions
- derivative definitions and cache locations
- sync pipeline settings (workers, batching)

Exact schemas and examples will be documented once the configuration stabilizes.

---

## Development status

- Early development
- Architecture and core abstractions are still evolving
- Breaking changes are expected
- No backward compatibility guarantees yet

This repository should be considered experimental.

---

## Building

```bash
go build ./...
```

---

## Running (example)

```bash
./lumenta --config ./config.yaml
```

Note: CLI flags, configuration structure, and runtime behavior are subject to change.

---

## Contributing

This is currently a personal project.

If you intend to experiment or contribute:

- focus on architectural clarity
- keep changes small and well-isolated
- do not introduce manual album curation paths
- prefer explicit, deterministic behavior over convenience

---

## License

License information will be added at a later stage, once the project structure stabilizes.
