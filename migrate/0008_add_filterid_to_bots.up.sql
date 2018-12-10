ALTER TABLE `bots` ADD filterid VARCHAR(36);
ALTER TABLE `bots` ADD INDEX `filterid_index` (`filterid`);
