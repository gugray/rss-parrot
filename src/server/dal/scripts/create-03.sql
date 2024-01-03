DELETE FROM toots WHERE account_id IN (SELECT id FROM accounts WHERE handle='');
DELETE FROM feed_posts WHERE account_id IN (SELECT id FROM accounts WHERE handle='');
DELETE FROM followers WHERE account_id IN (SELECT id FROM accounts WHERE handle='');
DELETE FROM accounts WHERE handle='';
