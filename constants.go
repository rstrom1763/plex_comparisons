package main

const PLEX_DB_PATH string = "/var/lib/plexmediaserver/Library/Application Support/Plex Media Server/Plug-in Support/Databases/com.plexapp.plugins.library.db"

const DUMP_QUERY string = `SELECT
    metadata_items.title,
    metadata_items.content_rating AS rating,
    IFNULL(year,0) AS year,
    metadata_items.tags_genre AS genre,
    library_sections.name AS library,
    CASE library_sections.section_type
        WHEN 1 THEN 'Movie'
        WHEN 2 THEN 'TV Show'
        WHEN 8 THEN 'Music'
        WHEN 13 THEN 'Photo'
        ELSE 'Unknown'
        END AS media_type,
    media_parts.file,
    media_parts.hash,
    media_parts.size,
    IFNULL(media_parts.duration,0) AS duration,
    media_items.container,
    IFNULL(media_items.bitrate,0) AS bitrate,
    media_items.video_codec,
    IFNULL(media_items.height,0) AS height,
    IFNULL(media_items.width,0) AS width,
    CASE
        WHEN width <= 640 THEN 'SD'
        WHEN width <= 1280 THEN '720p'
        WHEN width <= 2048 THEN '1080p'
        WHEN width <= 2560 THEN '1440p'
        ELSE '4K' END AS resolution,
    media_items.audio_codec

FROM metadata_items
    JOIN media_items ON media_items.metadata_item_id = metadata_items.id
    JOIN media_parts ON media_parts.media_item_id = media_items.id
    JOIN library_sections ON media_items.library_section_id = library_sections.id;`
