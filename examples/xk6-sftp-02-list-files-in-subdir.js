import sftp from 'k6/x/sftp';

const host = __ENV.SFTP_HOST;
const port = parseInt(__ENV.SFTP_PORT) || 22;
const user = __ENV.SFTP_USER;
const pass = __ENV.SFTP_PASS;
const remotePath = __ENV.SFTP_REMOTE_PATH || '/upload';

export default function () {
    let conn;
    try {
        conn = sftp.connect(host, user, pass, port);

        // ls() returns an array of objects with name, size, isDir, modTime properties
        const files = conn.ls(remotePath);

        files.forEach((file) => {
            const type = file.isDir ? 'DIR' : 'FILE';
            console.log(`[${type}] ${file.name} (${file.size} bytes)`);
        });
    } catch (err) {
        console.error(`SFTP error: ${err}`);
    } finally {
        if (conn) {
            conn.close();
        }
    }
}
