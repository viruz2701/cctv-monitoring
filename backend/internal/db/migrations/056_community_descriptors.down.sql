-- +migrate Down
-- PROTO-07: Community Protocol Registry — откат

DROP TRIGGER IF EXISTS trg_community_descriptors_updated ON community_descriptors;
DROP FUNCTION IF EXISTS trg_community_descriptors_updated();

DROP INDEX IF EXISTS idx_descriptor_ratings_descriptor;
DROP INDEX IF EXISTS idx_community_descriptors_verified;
DROP INDEX IF EXISTS idx_community_descriptors_downloads;
DROP INDEX IF EXISTS idx_community_descriptors_rating;
DROP INDEX IF EXISTS idx_community_descriptors_vendor;

DROP TABLE IF EXISTS community_descriptor_ratings;
DROP TABLE IF EXISTS community_descriptors;
