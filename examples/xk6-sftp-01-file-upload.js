import sftp from 'k6/x/sftp';

// Use binary mode ('b') to get ArrayBuffer, which converts cleanly to []byte in Go
const fileData = open('example.txt', 'b');

const host = __ENV.SFTP_HOST;
const port = parseInt(__ENV.SFTP_PORT) || 22;
const user = __ENV.SFTP_USER;
const pass = __ENV.SFTP_PASS;
const remotePath = __ENV.SFTP_REMOTE_PATH || '/upload/example.txt';

export default function () {
    let conn;
    try {
        conn = sftp.connect(host, user, pass, port);
        conn.upload(fileData, remotePath);
        console.log(`Uploaded to ${remotePath}`);
    } catch (err) {
        console.error(`SFTP error: ${err}`);
    } finally {
        if (conn) {
            conn.close();
        }
    }
}
