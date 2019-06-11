ALTER TABLE `chatgroups` ADD lastmsgid VARCHAR(64);
ALTER TABLE `chatgroups` ADD INDEX `lastmsgid_index` (`lastmsgid`);
