-- +goose Up
ALTER TABLE brand_social_link DROP CONSTRAINT brand_social_link_pkey;

ALTER TABLE brand_social_link DROP CONSTRAINT brand_social_link_brand_id_platform_key;

ALTER TABLE brand_social_link ADD CONSTRAINT brand_social_link_pkey PRIMARY KEY (brand_id, platform);

ALTER TABLE brand_social_link DROP COLUMN id;

UPDATE brand_working_hours SET is_closed = FALSE WHERE is_closed IS NULL;

ALTER TABLE brand_working_hours ALTER COLUMN is_closed SET NOT NULL;

-- +goose Down
ALTER TABLE brand_social_link ADD COLUMN id SERIAL;

ALTER TABLE brand_social_link DROP CONSTRAINT brand_social_link_pkey;

ALTER TABLE brand_social_link ADD CONSTRAINT brand_social_link_pkey PRIMARY KEY (id);

ALTER TABLE brand_social_link ADD CONSTRAINT brand_social_link_brand_id_platform_key UNIQUE (brand_id, platform);

ALTER TABLE brand_working_hours ALTER COLUMN is_closed DROP NOT NULL;
