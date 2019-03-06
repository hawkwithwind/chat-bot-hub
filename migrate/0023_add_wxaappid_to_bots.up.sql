ALTER TABLE `bots` ADD `wxaappid` VARCHAR(36);
ALTER TABLE `bots` ADD INDEX `wxaappid_index` (`wxaappid`);
