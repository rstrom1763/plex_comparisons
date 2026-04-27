package constants

const MOVIE_DUMP_QUERY string = `SELECT
    metadata_items.title,
    metadata_items.content_rating AS rating,
    IFNULL(year,0) AS year,
    metadata_items.tags_genre AS genre,
    library_sections.name AS library,
    CASE library_sections.section_type
        WHEN 1 THEN 'Movie'
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
    media_items.audio_codec,
    IFNULL(metadata_items.rating,0) AS critic_rating,
    IFNULL(metadata_items.audience_rating,0) AS audience_rating,
    metadata_items.hash AS metadata_hash
FROM metadata_items
    JOIN media_items ON media_items.metadata_item_id = metadata_items.id
    JOIN media_parts ON media_parts.media_item_id = media_items.id
    JOIN library_sections ON media_items.library_section_id = library_sections.id
WHERE section_type = 1
ORDER BY metadata_items.title;`

const SONG_DUMP_QUERY string = `SELECT
    metadata_items.title                         AS track_title,
    COALESCE(metadata_items.year, album.year, 0) AS year,
    metadata_items.tags_genre                    AS genre,
    album.title                                  AS album_title,
    artist.title                                 AS artist_name,
    library_sections.name                        AS library,
    CASE library_sections.section_type
        WHEN 1  THEN 'Movie'
        WHEN 2  THEN 'TV Show'
        WHEN 8  THEN 'Music'
        WHEN 13 THEN 'Photo'
        ELSE 'Unknown'
    END                                          AS media_type,
    media_parts.file,
    media_parts.hash,
    media_parts.size,
    IFNULL(media_parts.duration, 0)              AS duration,
    IFNULL(media_items.bitrate, 0)               AS bitrate,
    media_items.audio_codec,
    metadata_items.hash AS metadata_hash
FROM metadata_items
    JOIN media_items
        ON media_items.metadata_item_id = metadata_items.id
    JOIN media_parts
        ON media_parts.media_item_id = media_items.id
    JOIN library_sections
        ON media_items.library_section_id = library_sections.id

    LEFT JOIN metadata_items AS album
        ON album.id = metadata_items.parent_id
           AND album.metadata_type = 9

    LEFT JOIN metadata_items AS artist
        ON artist.id = album.parent_id
           AND artist.metadata_type = 8
WHERE
    library_sections.section_type = 8
    AND metadata_items.metadata_type = 10;`

const EPISODE_DUMP_QUERY string = `SELECT
    shows.title AS show_title,
    seasons."index" AS season_number,
    episodes."index" AS episode_number,
    episodes.title AS episode_title,
    episodes.content_rating,
    IFNULL(episodes.year,0) AS year,
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
    media_items.audio_codec,
    IFNULL(shows.rating,0) AS critic_rating,
    IFNULL(shows.audience_rating,0) AS audience_rating,
    episodes.hash AS metadata_hash
FROM metadata_items AS episodes
    JOIN metadata_items AS seasons ON episodes.parent_id = seasons.id
    JOIN metadata_items AS shows ON seasons.parent_id = shows.id
    JOIN media_items ON media_items.metadata_item_id = episodes.id
    JOIN media_parts ON media_parts.media_item_id = media_items.id
    JOIN library_sections ON media_items.library_section_id = library_sections.id
WHERE episodes.metadata_type = 4
  AND seasons.metadata_type = 3
  AND shows.metadata_type = 2
ORDER BY show_title, season_number, episode_number;`
