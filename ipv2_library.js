
import { S3Client } from 'https://jslib.k6.io/aws/0.14.0/s3.js';
import { AWSConfig, SQSClient } from 'https://jslib.k6.io/aws/0.14.0/sqs.js';
import sftp from 'k6/x/sftp';

/**
 * Upload a file to a remote SFTP endpoint
**/
export async function uploadFileToSftp(sftpHost, sftpUsername, sftpPassword, sftpPort, fileDataToUpload, fileNameToWrite) {
  let connection;

  try {
    connection = sftp.connect(sftpHost, sftpUsername, sftpPassword, sftpPort);

    connection.upload(fileDataToUpload, fileNameToWrite);
    
    console.debug(`Uploaded file: ${fileNameToWrite}`);
  } catch (err) {
      fail(err);
  } finally {
    if (connection) {
      connection.close();
    }
  }
}

export async function sendDecryptionRequest(queue, transactionId, sourcePath, destPath, organization, interfaceId, outputQueue) {
  try {
    const decryptMessage = {
      "transactionId": transactionId,
      "type": "DECRYPTION",
      "sourcePath": sourcePath,
      "destinationPath": destPath,
      "organization": organization,
      "interfaceId": interfaceId,
      "outputQueue": outputQueue
    };

    const decryptRequestDeliveryResponse = await sqs.sendMessage(queue, JSON.stringify(decryptMessage), {messageGroupId: "1234"});
    console.debug(`[${transactionId}] Sent DECRYPTION request`);
  } catch (err) {
    console.error(`Error in sending SQS decryption message: ${err}`)
    fail(err)
  }
}
