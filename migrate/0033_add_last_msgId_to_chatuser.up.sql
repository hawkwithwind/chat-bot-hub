ALTER TABLE `chatusers` ADD lastmsgid VARCHAR(64);
ALTER TABLE `chatusers` ADD INDEX `lastmsgid_index` (`lastmsgid`);
