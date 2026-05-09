-- =========================================================
-- USERS
-- =========================================================

CREATE TABLE IF NOT EXISTS users (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY 
    COMMENT 'Internal unique user identifier',
  username VARCHAR(100) NOT NULL UNIQUE 
    COMMENT 'Unique login name used for authentication',
  pass_hash VARCHAR(255) NOT NULL 
    COMMENT 'Password hash (bcrypt / argon2 / similar)',
  email VARCHAR(255) NULL 
    COMMENT 'Optional contact email address',
  role ENUM('user','admin') NOT NULL DEFAULT 'user' 
    COMMENT 'High-level role for administrative privileges',
  disabled BOOLEAN NOT NULL DEFAULT FALSE 
    COMMENT 'Soft-disable flag; disabled users cannot log in',
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP 
    COMMENT 'Account creation timestamp'
) ENGINE=InnoDB COMMENT='Application users';

-- =========================================================
-- IMAGES
-- =========================================================

CREATE TABLE IF NOT EXISTS images (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY
    COMMENT 'Image unique identifier',
  root VARCHAR(50) NOT NULL
    COMMENT 'Name of the configured root',
  path VARCHAR(600) NOT NULL
    COMMENT 'Directory path relative to configured root',
  filename VARCHAR(64) NOT NULL
    COMMENT 'Filename without extension',
  ext VARCHAR(8) NOT NULL
    COMMENT 'File extension',

  file_size INT UNSIGNED NOT NULL
    COMMENT 'File size in bytes',
  mtime DATETIME NOT NULL 
    COMMENT 'Filesystem modification timestamp',
  file_hash CHAR(64) NOT NULL 
    COMMENT 'SHA-256 hash of file',
  meta_hash CHAR(64) NOT NULL 
    COMMENT 'SHA-256 hash of sidecar file',

  taken_at DATETIME NULL
    COMMENT 'Photo capture timestamp (EXIF)',
  camera VARCHAR(128) NULL 
    COMMENT 'Camera model',
  lens VARCHAR(128) NULL 
    COMMENT 'Lens model',
  focal_length DECIMAL(5,1) NULL 
    COMMENT 'Focal length in millimeters',
  aperture DECIMAL(4,1) NULL
    COMMENT 'Aperture (f-number)',
  exposure DECIMAL(8,6) NULL
    COMMENT 'Exposure time in seconds',
  iso SMALLINT UNSIGNED NULL
    COMMENT 'ISO sensitivity value',

  latitude DOUBLE NULL
    COMMENT 'GPS latitude (WGS84)',
  longitude DOUBLE NULL
    COMMENT 'GPS longitude (WGS84)',

  rotation SMALLINT NULL
    COMMENT 'Image rotation / orientation',
  rating SMALLINT NULL
    COMMENT 'User or source rating value',

  width INT UNSIGNED NOT NULL
    COMMENT 'Image width (px)',
  height INT UNSIGNED NOT NULL
    COMMENT 'Image height (px)',
  panorama TINYINT NOT NULL
    COMMENT '1 if marked panorama',

  title VARCHAR(255) NULL 
    COMMENT 'Human-readable image title',
  caption TEXT NULL 
    COMMENT 'Longer image description',

  focus_x FLOAT NULL
    COMMENT 'Normalized horizontal focus point (0..1)',
  focus_y FLOAT NULL
    COMMENT 'Normalized vertical focus point (0..1)',
  focus_mode ENUM('auto','manual','center','top','bottom','left','right') DEFAULT 'auto'
    COMMENT 'Focus point selection mode',

  exif_json JSON NULL
    COMMENT 'EXIF/XMP metadata dump in key-value format',

  acl_level INT NOT NULL DEFAULT 0
    COMMENT 'Final image-level access control level',
  acl_user_id BIGINT UNSIGNED DEFAULT 0
    COMMENT 'User ID for user-level access 0: no user assigned',

  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    COMMENT 'Image record creation timestamp',
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
    COMMENT 'Last update timestamp',

  last_seen_sync BIGINT UNSIGNED NULL
    COMMENT 'Sync run ID in which the image was last observed',

  order_date DATETIME NOT NULL
    COMMENT 'Navigation sort key (taken_at fallback date_available). Non-NULL. Used in ORDER BY (order_date, filename, id).',

  UNIQUE KEY uniq_image_path (root, path, filename, ext),
  INDEX idx_images_taken_at (taken_at),
  INDEX idx_images_camera (camera),
  INDEX idx_images_lens (lens),
  INDEX idx_images_gps (latitude, longitude),
  INDEX idx_images_acl (acl_level, acl_user_id),
  INDEX idx_images_last_seen_sync (last_seen_sync),
  INDEX idx_images_order_acl (acl_level, acl_user_id, order_date, filename, id),
  INDEX idx_images_order (order_date, filename, id),
  INDEX idx_hash_order (acl_level, acl_user_id, file_hash, id),
  INDEX ixd_last_seen (last_seen_sync)
) ENGINE=InnoDB COMMENT='Images with filesystem identity, metadata, and ACL';


-- =========================================================
-- FILTERED OUT IMAGES
-- =========================================================

CREATE TABLE IF NOT EXISTS filtered (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY
    COMMENT 'Image unique identifier',
  root VARCHAR(50) NOT NULL
    COMMENT 'Name of the configured root',
  path VARCHAR(600) NOT NULL
    COMMENT 'Directory path relative to configured root',
  filename VARCHAR(64) NOT NULL
    COMMENT 'Filename without extension',
  ext VARCHAR(8) NOT NULL
    COMMENT 'File extension',

  file_size INT UNSIGNED NOT NULL
    COMMENT 'File size in bytes',
  mtime DATETIME NOT NULL 
    COMMENT 'Filesystem modification timestamp',
  file_hash CHAR(64) NOT NULL 
    COMMENT 'SHA-256 hash of file',
  meta_hash CHAR(64) NOT NULL 
    COMMENT 'SHA-256 hash of sidecar file',

  exif_json JSON NULL
    COMMENT 'EXIF/XMP metadata dump in key-value format',

  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    COMMENT 'Image record creation timestamp',
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
    COMMENT 'Last update timestamp',

  last_seen_sync BIGINT UNSIGNED NULL
    COMMENT 'Sync run ID in which the image was last observed',

  UNIQUE KEY uniq_image_path (root, path, filename, ext),
  INDEX idx_images_last_seen_sync (last_seen_sync),
  INDEX ixd_last_seen (last_seen_sync)
) ENGINE=InnoDB COMMENT='Filesystem images excluded from the library during sync';



-- =========================================================
-- ALBUMS (categories)
-- =========================================================

CREATE TABLE IF NOT EXISTS albums (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY 
    COMMENT 'Album unique identifier',
  parent_id BIGINT UNSIGNED NULL 
    COMMENT 'Parent album ID (NULL for root albums)',
  name VARCHAR(255) NOT NULL 
    COMMENT 'Display name of the album',
  description TEXT NULL 
    COMMENT 'Optional longer album description',
  ancestor_ids JSON NOT NULL 
    COMMENT 'Materialized path: ordered list of ancestor album IDs including self',
  rule_json JSON NOT NULL 
    COMMENT 'Serialized dynamic rule tree defining album contents',
  rank INT NOT NULL DEFAULT 0 
    COMMENT 'Sibling ordering index within the same parent album',
  cover_image_id BIGINT UNSIGNED NULL 
    COMMENT 'Optional fixed cover image overriding automatic selection',
  child_album_count INT UNSIGNED NOT NULL DEFAULT 0 
    COMMENT 'Cached recursive count of child albums',
  image_count INT UNSIGNED NOT NULL DEFAULT 0
    COMMENT 'Cached recursive count of images in this album',
  acl_level INT NOT NULL DEFAULT 0
    COMMENT 'Album-level access control level',
  acl_user_id BIGINT UNSIGNED DEFAULT 0
    COMMENT 'User ID for user-level album access, 0: not set',
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
    COMMENT 'Last modification timestamp',
  FOREIGN KEY (parent_id) REFERENCES albums(id) ON DELETE CASCADE,
  FOREIGN KEY (cover_image_id) REFERENCES images(id) ON DELETE SET NULL,

  INDEX idx_albums_parent (parent_id),
  INDEX idx_albums_acl (acl_level, acl_user_id),
  INDEX idx_albumc_cover (cover_image_id)
) ENGINE=InnoDB COMMENT='Hierarchical albums with dynamic rule-based contents';

-- =========================================================
-- ALBUM <-> IMAGE RELATION
-- =========================================================

CREATE TABLE IF NOT EXISTS album_images (
  album_id BIGINT UNSIGNED NOT NULL
    COMMENT 'Referenced album ID',
  image_id BIGINT UNSIGNED NOT NULL
    COMMENT 'Referenced image ID',

  position INT UNSIGNED NULL 
    COMMENT 'Optional ordering position inside album',
  computed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    COMMENT 'Timestamp when album rules were evaluated',

  PRIMARY KEY (album_id, image_id),

  FOREIGN KEY (album_id) REFERENCES albums(id) ON DELETE CASCADE,
  FOREIGN KEY (image_id) REFERENCES images(id) ON DELETE CASCADE,

  INDEX idx_album_images_album_pos (album_id, position),
  INDEX idx_album_images_image (image_id)
) ENGINE=InnoDB COMMENT='Materialized album-to-image assignments';


-- =========================================================
-- TAGS
-- =========================================================

CREATE TABLE IF NOT EXISTS tags (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY
    COMMENT 'Tag unique identifier',

  name VARCHAR(100) NOT NULL
    COMMENT 'Single segment tag name',
  parent_id BIGINT NULL
    COMMENT 'Parent tag ID for hierarchy',

  source ENUM('digikam') NOT NULL DEFAULT 'digikam'
    COMMENT 'Origin of the tag taxonomy',

  UNIQUE KEY uniq_parent_name (parent_id, name),
  INDEX idx_tags_name (name)
) ENGINE=InnoDB COMMENT='Hierarchical read-only tags imported from digiKam';

-- =========================================================
-- IMAGE <-> TAG RELATION
-- =========================================================

CREATE TABLE IF NOT EXISTS image_tags (
  image_id BIGINT UNSIGNED NOT NULL
    COMMENT 'Referenced image ID',
  tag_id BIGINT UNSIGNED NOT NULL
    COMMENT 'Referenced tag ID',

  PRIMARY KEY (image_id, tag_id),

  FOREIGN KEY (image_id) REFERENCES images(id) ON DELETE CASCADE,
  FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE,

  INDEX idx_image_tags_tag (tag_id, image_id)
) ENGINE=InnoDB COMMENT='Assignment of tags to images';

-- =========================================================
-- SYNC RUNS
-- =========================================================

CREATE TABLE IF NOT EXISTS sync_runs (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY
    COMMENT 'Sync run unique identifier',

  is_active TINYINT NULL DEFAULT 1
    COMMENT '1 = currently active run, NULL = historical run (unique index ensures only one active)',

  started_at DATETIME NOT NULL
    COMMENT 'Sync start timestamp',
  finished_at DATETIME NULL
    COMMENT 'Sync completion timestamp',

  mode TEXT(50) NOT NULL DEFAULT 'full'
    COMMENT 'Execution mode of the sync run',

  total_seen INT UNSIGNED NOT NULL DEFAULT 0
    COMMENT 'Number of images observed during sync',
  total_deleted INT UNSIGNED NOT NULL DEFAULT 0
    COMMENT 'Number of images removed during sync',
    
  status ENUM('running','finished','failed') NOT NULL DEFAULT 'running'
    COMMENT 'Final execution status of the sync run',

  error TEXT NULL COMMENT 'Optional error details if sync failed',

  meta_hash CHAR(64) NULL
    COMMENT 'Hash of the used metadata settings',

  UNIQUE KEY uniq_sync_active (is_active),
  INDEX idx_sync_runs_started (started_at),
  INDEX idx_sync_runs_success (status, started_at),  
  INDEX idx_sync_runs_status (status)
) ENGINE=InnoDB COMMENT='Filesystem synchronization runs and diagnostics';


-- =========================================================
-- FILE SYNC RUNS
-- =========================================================

CREATE TABLE IF NOT EXISTS sync_files (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,

  sync_id BIGINT UNSIGNED NOT NULL,

  root     VARCHAR(50)  NOT NULL,
  path     VARCHAR(600) NOT NULL,
  filename VARCHAR(64)  NOT NULL,
  ext      VARCHAR(8)   NOT NULL,

  status VARCHAR(50) NOT NULL,
  dirty_reason VARCHAR(64) NULL,

  ruleresults_json   JSON NULL,

  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

  INDEX idx_sync_files_sync (sync_id),
  INDEX idx_sync_files_file_history (root, path, filename, ext),
  INDEX idx_sync_files_sync_path ( sync_id, root, path, filename, ext),
  INDEX idx_sync_files_order (root, path, filename, ext, created_at DESC),
  CONSTRAINT fk_sync_files_run
    FOREIGN KEY (sync_id) REFERENCES sync_runs(id)
    ON DELETE CASCADE
) ENGINE=InnoDB ;