package archive

// maxExtractedFileSize limits decompressed members to prevent archive bombs
// from exhausting disk space during installation.
const maxExtractedFileSize int64 = 512 * 1024 * 1024
