ALTER TABLE `chatusers` ADD `lastsendat` DATETIME;
ALTER TABLE `chatusers` ADD INDEX `lastsendat_index` (`lastsendat`);

ALTER TABLE `chatgroups` ADD `lastsendat` DATETIME;
ALTER TABLE `chatgroups` ADD INDEX `lastsendat_index` (`lastsendat`);


