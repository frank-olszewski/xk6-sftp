import sftp from 'k6/x/sftp';

const host = __ENV.SFTP_HOST;
const port = parseInt(__ENV.SFTP_PORT) || 22;
const user = __ENV.SFTP_USER;
const pass = __ENV.SFTP_PASS;
const remotePath = __ENV.SFTP_REMOTE_FILE || '/upload/example.txt';
const localPath = __ENV.SFTP_LOCAL_FILE || '/tmp/downloaded.txt';

export default function () {
    let conn;
    try {
        conn = sftp.connect(host, user, pass, port);
        conn.download(remotePath, localPath);
        console.log(`Downloaded ${remotePath} to ${localPath}`);
    } catch (err) {
        console.error(`SFTP error: ${err}`);
    } finally {
        if (conn) {
            conn.close();
        }
    }
}
