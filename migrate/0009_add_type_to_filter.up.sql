ALTER TABLE `filters` ADD filtertype VARCHAR(64) NOT NULL;
ALTER TABLE `filters` ADD INDEX `filtertype_index` (`filtertype`);
