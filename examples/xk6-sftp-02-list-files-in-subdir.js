import sftp from 'k6/x/sftp';

const sftp_host = __ENV.SFTP_HOST;
const sftp_port = __ENV.SFTP_PORT;
const sftp_user = __ENV.SFTP_USER;
const sftp_pass = __ENV.SFTP_PASS;

export default function() {
    const remotePath = "./dropoff";

    sftp.connect(sftp_host, sftp_user, sftp_pass, sftp_port);
    let folderContents = sftp.ls(remotePath);
    sftp.disconnect();

    folderContents.forEach((file) => console.log(` [${file.isDir() ? "DIR" : "FIL"}] ${remotePath}${file.name()}  .. ${file.size()}`));
}
