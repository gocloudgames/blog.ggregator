-- name: GetFeedFollowsForUser :many
SELECT
    ff.*,
    u.name AS user_name,
    f.name AS feed_name
FROM feed_follows ff
JOIN users u ON ff.user_id = u.id
JOIN feeds f ON ff.feed_id = f.id
WHERE ff.user_id = $1;

-- name: DeleteFeedFollowByUserAndFeedURL :exec
DELETE FROM feed_follows
WHERE feed_follows.user_id = $1
  AND feed_follows.feed_id = (
    SELECT id FROM feeds WHERE url = $2
  );