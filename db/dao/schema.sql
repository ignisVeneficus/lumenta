-- =========================================================
-- USERS
-- =========================================================

CREATE TABLE IF NOT EXISTS  users (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  username VARCHAR(100) NOT NULL UNIQUE,
  pass_hash VARCHAR(255) NOT NULL,
  email VARCHAR(255),
  role ENUM('user','admin') NOT NULL DEFAULT 'user',
  disabled BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB;

-- =========================================================
-- GROUPS (előre tervezve, de nem kötelező használni)
-- =========================================================

CREATE TABLE IF NOT EXISTS  groups (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  name VARCHAR(100) NOT NULL UNIQUE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS  user_groups (
  user_id BIGINT UNSIGNED NOT NULL,
  group_id BIGINT UNSIGNED NOT NULL,
  PRIMARY KEY (user_id, group_id),
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE
) ENGINE=InnoDB;

-- =========================================================
-- ALBUMS (categories)
-- =========================================================

CREATE TABLE IF NOT EXISTS  albums (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  parent_id BIGINT UNSIGNED NULL,

  name VARCHAR(255) NOT NULL,

  description TEXT NULL,


  -- materialized path JSON formában
  -- root -> ... -> self (SAJÁT ID BENNE VAN)
  ancestor_ids JSON NOT NULL,

  -- dinamikus album szabály (AND/OR/NOT fa)
  rule_json JSON NOT NULL,


  -- sibling order
  rank INT NOT NULL DEFAULT 0,

  -- opcionális fix borító
  cover_image_id BIGINT UNSIGNED NULL,


  -- UI cache mezők (rekurzív)
  child_album_count INT UNSIGNED NOT NULL DEFAULT 0,
  image_count       INT UNSIGNED NOT NULL DEFAULT 0,

  -- album ACL (navigációs szint)
  acl_scope ENUM('public','any_user','user','group','admin') NOT NULL DEFAULT 'public',
  acl_user_id BIGINT UNSIGNED NULL,
  acl_group_id BIGINT UNSIGNED NULL,

  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

  FOREIGN KEY (parent_id) REFERENCES albums(id) ON DELETE CASCADE,

  INDEX idx_albums_parent (parent_id),
  INDEX idx_albums_acl_user (acl_user_id),
  INDEX idx_albums_acl_group (acl_group_id)
) ENGINE=InnoDB;

-- =========================================================
-- IMAGES
-- =========================================================

CREATE TABLE IF NOT EXISTS  images (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,

  -- path felbontva (root configból jön)
  path VARCHAR(384) NOT NULL,
  filename VARCHAR(64) NOT NULL,
  ext VARCHAR(8) NOT NULL,

  -- filesystem fingerprint
  file_size INT UNSIGNED NOT NULL,
  mtime DATETIME NOT NULL,
  file_hash CHAR(64) NOT NULL,   -- sha256
  meta_hash CHAR(64) NOT NULL,   -- exif/xmp kanonikus hash

  -- gyakran használt EXIF mezők
  taken_at DATETIME NULL,
  camera VARCHAR(128) NULL,
  lens VARCHAR(128) NULL,
  focal_length DECIMAL(5,1) NULL,
  aperture DECIMAL(4,1) NULL,
  exposure DECIMAL(8,6) NULL,
  iso SMALLINT UNSIGNED NULL,


  latitude DOUBLE NULL,
  longitude DOUBLE NULL,

  rotation SMALLINT NULL,
  rating SMALLINT NULL,
  
  -- emberi megnevezés / leírás
  title VARCHAR(255) NULL,
  subject TEXT NULL,
  
  -- fókuszpont (crop-aware thumbnails)
  focus_x FLOAT NULL, -- 0..1
  focus_y FLOAT NULL, -- 0..1
  focus_mode ENUM('auto','manual','center','top','bottom') DEFAULT 'auto',

  -- teljes EXIF / XMP dump
  exif_json JSON NULL,

  -- kép ACL (végső döntés)
  acl_scope ENUM('public','any_user','user','group','admin') NOT NULL DEFAULT 'public',
  acl_user_id BIGINT UNSIGNED NULL,
  acl_group_id BIGINT UNSIGNED NULL,

  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

  last_seen_sync BIGINT UNSIGNED NULL COMMENT
  'ID of the last sync_runs entry in which this image was seen',

  UNIQUE KEY uniq_image_path (path, filename, ext),
  INDEX idx_images_taken_at (taken_at),
  INDEX idx_images_camera (camera),
  INDEX idx_images_lens (lens),
  INDEX idx_images_gps (latitude, longitude),
  INDEX idx_images_acl_user (acl_user_id),
  INDEX idx_images_acl_group (acl_group_id),
  INDEX idx_images_last_seen_sync (last_seen_sync),
) ENGINE=InnoDB;

-- =========================================================
-- ALBUM <-> IMAGE kapcsolat (materializált rule engine eredmény)
-- =========================================================

CREATE TABLE IF NOT EXISTS  album_images (
  album_id BIGINT UNSIGNED NOT NULL,
  image_id BIGINT UNSIGNED NOT NULL,

  position INT UNSIGNED NULL, -- sort / rank
  computed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

  PRIMARY KEY (album_id, image_id),

  FOREIGN KEY (album_id) REFERENCES albums(id) ON DELETE CASCADE,
  FOREIGN KEY (image_id) REFERENCES images(id) ON DELETE CASCADE,

  INDEX idx_album_images_album_pos (album_id, position),
  INDEX idx_album_images_image (image_id)
  
) ENGINE=InnoDB;

-- =========================================================
-- TAGS (hierarchikus, read-only, Digikam-ból)
-- =========================================================

CREATE TABLE IF NOT EXISTS tags (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  name VARCHAR(100) NOT NULL,
  parent_id BIGINT NULL,
  source ENUM('digikam') NOT NULL DEFAULT 'digikam',

  UNIQUE KEY uniq_parent_name (parent_id, name),
  INDEX idx_tags_name (name)
) ENGINE=InnoDB;

-- =========================================================
-- IMAGE <-> TAG kapcsolat
-- =========================================================

CREATE TABLE IF NOT EXISTS image_tags (
  image_id BIGINT UNSIGNED NOT NULL,
  tag_id BIGINT UNSIGNED NOT NULL,

  PRIMARY KEY (image_id, tag_id),

  FOREIGN KEY (image_id) REFERENCES images(id) ON DELETE CASCADE,
  FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE,

  INDEX idx_image_tags_tag (tag_id, image_id)

) ENGINE=InnoDB;

-- =========================================================
-- SYNC ADMIN kapcsolat
-- =========================================================

CREATE TABLE IF NOT EXISTS sync_runs (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  is_active TINYINT NULL DEFAULT 1,

  started_at DATETIME NOT NULL,
  finished_at DATETIME NULL,

  mode ENUM('full','incremental','partial','dry-run') NOT NULL DEFAULT 'full',

  total_seen INT UNSIGNED NOT NULL DEFAULT 0,
  total_deleted INT UNSIGNED NOT NULL DEFAULT 0,

  status ENUM('running','finished','failed') NOT NULL DEFAULT 'running',

  error TEXT NULL,

  UNIQUE KEY uniq_sync_active (is_active),
  INDEX idx_sync_runs_started (started_at),
  INDEX idx_sync_runs_status (status)
) ENGINE=InnoDB;