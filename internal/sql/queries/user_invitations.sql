-- name: CreateUserInvitation :exec
INSERT INTO user_invitations (token, user_id, expiry)
VALUES ($1,$2,$3);

-- name: DeleteUserInvitation :exec
DELETE FROM user_invitations
WHERE user_id = $1;

-- name: GetUserFromInvitation :one
SELECT u.id FROM users u JOIN user_invitations ui
ON u.id = ui.user_id
WHERE ui.token = $1 AND ui.expiry > NOW()
;
