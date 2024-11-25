import sftp from 'k6/x/sftp';

const file_to_upload = open("example.txt");
const sftp_host = __ENV.SFTP_HOST;
const sftp_port = __ENV.SFTP_PORT;
const sftp_user = __ENV.SFTP_USER;
const sftp_pass = __ENV.SFTP_PASS;

export default function() {
    console.log(`Connecting to ${sftp_user}@${sftp_host}:${sftp_port}`)
    sftp.connect(sftp_host, sftp_user, sftp_pass, sftp_port);
    sftp.upload(file_to_upload, "example.txt");
    sftp.disconnect();
}
