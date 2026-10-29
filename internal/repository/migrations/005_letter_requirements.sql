UPDATE jenis_surat SET syarat=ARRAY['KTP pemohon','KK','Surat pengantar RT'] WHERE kode='DM' AND cardinality(syarat)=0;
UPDATE jenis_surat SET syarat=ARRAY['KTP pemohon','KK','Surat pengantar RT'] WHERE kode='SU' AND cardinality(syarat)=0;
UPDATE jenis_surat SET syarat=ARRAY['KTP pemohon','KK','Surat pengantar RT'] WHERE kode='TM' AND cardinality(syarat)=0;
UPDATE jenis_surat SET syarat=ARRAY['KTP almarhum','KK almarhum','Surat pengantar RT','KTP pelapor'] WHERE kode='KM' AND cardinality(syarat)=0;
