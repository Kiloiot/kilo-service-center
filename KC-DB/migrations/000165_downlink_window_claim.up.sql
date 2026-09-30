-- Record whether a telegram's downlink window has been claimed.
--
-- An endpoint opens one downlink window per uplink (radio §3.6.1), and every
-- base station that received the telegram reports it. The first reception
-- through a bidirectional base station claims the window for the downlink it
-- dispatches, so a later reception can never queue a second downlink for the
-- same window, while a reception through a receive-only station leaves the
-- window to the next bidirectional one. A dispatch that sends nothing gives
-- the claim back. The archive mirrors the live layout (000139).

ALTER TABLE messages ADD COLUMN dl_window_claimed BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE messages_archive ADD COLUMN dl_window_claimed BOOLEAN NOT NULL DEFAULT false;

COMMENT ON COLUMN messages.dl_window_claimed IS
    'True while a reception of the telegram holds its downlink window for a dispatch (radio §3.6.1: one downlink per window)';
