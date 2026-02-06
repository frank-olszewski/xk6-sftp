import exec from 'k6/execution';
import { Trend } from 'k6/metrics';
import { sleep, check, fail } from 'k6';
import sftp from 'k6/x/sftp';
import { AWSConfig, SQSClient } from 'https://jslib.k6.io/aws/0.14.0/sqs.js'
import { S3Client } from 'https://jslib.k6.io/aws/0.14.0/s3.js'
import { randomIntBetween } from 'https://jslib.k6.io/k6-utils/1.2.0/index.js';
import { uploadFileToSftp } from './ipv2_library.js';

//const fileData=open("C3532_2025-11-03T14_43_13.txt.pgp", "b");
const fileData = open("mockb-2026-02-03.pri.gpg", "b");

const awsConfig = new AWSConfig ({
  region: "us-east-1",
  accessKeyId: __ENV.AWS_ACCESS_KEY_ID,
  secretAccessKey: __ENV.AWS_SECRET_ACCESS_KEY,
  sessionToken: __ENV.AWS_SESSION_TOKEN,
});


const newFileTrend = new Trend('time_to_new_file_msg', true);
const decryptTrend = new Trend('time_to_decryption_complete', true);
const encryptTrend = new Trend('time_to_encryption_complete', true);
const s3deliveryTrend = new Trend('time_to_send_file_s3', true);
const username = "mes-ip-test-user1";

const sqs = new SQSClient(awsConfig);
const s3 = new S3Client(awsConfig);
const newFileQueue="https://sqs.us-east-1.amazonaws.com/388433153595/dch-mes-mockba-dev-file.fifo";
const decryptRequestQueue="https://sqs.us-east-1.amazonaws.com/388433153595/dch-mes-mockba-dev-decrypt.fifo";
const decryptResponseQueue="https://sqs.us-east-1.amazonaws.com/388433153595/dch-mes-mockba-dev-decrypt-complete.fifo";
const encryptRequestQueue="https://sqs.us-east-1.amazonaws.com/388433153595/dch-mes-mockba-dev-encrypt.fifo";
const encryptResponseQueue="https://sqs.us-east-1.amazonaws.com/388433153595/dch-mes-mockba-dev-encrypt-complete.fifo";
const sendFileQueue="https://sqs.us-east-1.amazonaws.com/388433153595/dch-mes-mockba-dev-file-outbound.fifo";
const localDeliveryBucketName="dch-mes-mockb-dev-sftp-mes-ip-test-user1";

/*
export async function uploadFileToTransfer(fileDataToUpload, filenameToWriteAs) {
  let connection;
  try {
    connection=sftp.connect("sftp.mockb.gamest.cloud", "mes-ip-test-user1", "", 22);
  
    connection.upload(fileDataToUpload, filenameToWriteAs);
    console.debug(`Uploaded file: ${filenameToWriteAs}!`)
  } catch (err) {
    console.error(`Error: ${err}`);
    fail(err);
  } finally {
    if (connection) {
      connection.close();
    }
  }
}
*/
export async function waitForNewFileMessage(queue, filenameToWaitFor) {
  let continueWaiting = true;
  let newFilePath = undefined;
  console.debug(`Starting to wait for ${filenameToWaitFor} NEW_FILE message`);
  while (continueWaiting) {
    try {
      const messagesReceivedFromAws = await sqs.receiveMessages(queue, undefined, undefined, 10, 5, 2);

      for (let i = 0; i < messagesReceivedFromAws.length; i++) {
        if (messagesReceivedFromAws[i].Body !== undefined && messagesReceivedFromAws[i].Body !== "") {
          let messageBodyJson = JSON.parse(messagesReceivedFromAws[i].Body);
          if (messageBodyJson.type === "NEW_FILE" && messageBodyJson.data.originalS3Path.endsWith(`/${filenameToWaitFor}`)) {
            newFilePath = messageBodyJson.destination.s3Path;

            await sqs.deleteMessage(queue, messagesReceivedFromAws[i].ReceiptHandle);
            continueWaiting=false;
          }
        }
      }
    } catch (err) {
      console.error(`Error while waiting for new file message: ${err}`);
      fail(err)
    }
    if (continueWaiting) {
      console.debug(`Did not find ${filenameToWaitFor} NEW_FILE message, waiting before checking again...`)
      sleep(randomIntBetween(1,5));
    }
  }

  console.debug(`Found NEW_FILE message for ${filenameToWaitFor}`);
  return newFilePath;
}


export async function waitForDecryptResponseMessage(queue, transactionId) {
  let continueWaiting=true;
  while (continueWaiting) {
    try {
      const messagesReceivedFromAws = await sqs.receiveMessages(queue, undefined, undefined, 10, 5, 2);
      for (let i = 0; i < messagesReceivedFromAws?.length; i++) {

        let messageBodyJson = JSON.parse(messagesReceivedFromAws[i]?.Body);

        if (messageBodyJson.hasOwnProperty("error")) {
          fail(`Received error on decryption: ${messageBodyJson?.error?.message}`);
        } else {
          if (messageBodyJson?.request?.transactionId === transactionId) {
            await sqs.deleteMessage(queue, messagesReceivedFromAws[i].ReceiptHandle);
            continueWaiting = false;
          }
        }
      }
    } catch (err) {
      console.error(`Error in waiting for SQS decryption response: ${err}`);
      fail(err);
    }
  }
  console.debug(`[${transactionId}] Found DECRYPTION response message`);
}


export async function sendEncryptionRequest(queue, transactionId, sourcePath, destPath, organization, interfaceId, outputQueue) {
  try {
    const encryptMessage = {
      "transactionId": transactionId,
      "type": "ENCRYPTION",
      "sourcePath": sourcePath,
      "destinationPath": destPath,
      "organization": organization,
      "interfaceId": interfaceId,
      "outputQueue": outputQueue
    };

    const encryptRequestDeliveryResponse = await sqs.sendMessage(queue, JSON.stringify(encryptMessage), {messageGroupId: "1234"});
    console.debug(`[${transactionId}] Sent ENCRYPTION request`);
  } catch (err) {
    console.error(`Error in sending SQS encryption message: ${err}`)
    fail(err)
  }
}


export async function waitForEncryptResponseMessage(queue, transactionId) {
  let continueWaiting=true;
  while (continueWaiting) {
    try {
      const messagesReceivedFromAws = await sqs.receiveMessages(queue, undefined, undefined, 10, 5, 2);
      for (let i = 0; i < messagesReceivedFromAws?.length; i++) {

        let messageBodyJson = JSON.parse(messagesReceivedFromAws[i]?.Body);

        if (messageBodyJson.hasOwnProperty("error")) {
          fail(`Received error on encryption: ${messageBodyJson?.error?.message}`);
        } else {
          if (messageBodyJson?.request?.transactionId === transactionId) {
            await sqs.deleteMessage(queue, messagesReceivedFromAws[i].ReceiptHandle);
            continueWaiting = false;
          }
        }
      }
    } catch (err) {
      console.error(`Error in waiting for SQS encryption response: ${err}`);
      fail(err);
    }
  }
  console.debug(`[${transactionId}] Found ENCRYPTION response message`);
}

export async function sendFileRequest(queue, transactionId, sourcePath, recipient) {
  try {
    const sendFileRequest = {
      "transactionId": transactionId,
      "type": "SEND_FILE",
      "sourcePath": sourcePath,
      "targetPath": "loadtestoutgoing/",
      "recipient": recipient
    };

    const sendFileRequestDeliveryResponse = await sqs.sendMessage(queue, JSON.stringify(sendFileRequest), {messageGroupId: "1234"});
    console.debug(`[${transactionId}] Sent SEND_FILE request`);
  }  catch (err) {
    fail(err);
  }
}

export async function waitForS3FilePresence(bucketName, prefix, fileNameToCheckFor) {
  let continueWaiting=true;
  while (continueWaiting) {
    try {
      const objectsInBucket = await s3.listObjects(bucketName, prefix);
      const filteredObjectsInBucket = objectsInBucket.filter((o) => o.key.endsWith(`/${fileNameToCheckFor}`));

      if (filteredObjectsInBucket.length > 0) {
        return true;
      }
      else { 
        console.debug(`File [${fileNameToCheckFor}] was not found across ${objectsInBucket.length} object(s)`);
        for (let i = 0; i < objectsInBucket.length; i++) {
          console.debug(`File name returned: ${objectsInBucket[i].key}`);
        }
        sleep(randomIntBetween(1,3));
      }
    } catch (err) {
      fail(err);
    }
  }
}

export default async function() {
  const transactionIdToUseForVu = `${randomIntBetween(111111111,999999999)}-LOAD-${randomIntBetween(1111,9999)}-${randomIntBetween(1111,9999)}`;

  const uniqueSysFilename=`eld_gdc_daily_inbound_${randomIntBetween(10000,99999)}.txt.pgp`;

  const decryptDestPath=`s3://dch-mes-mockb-dev-files-ec89/mestest/dec_${uniqueSysFilename.slice(0,-4)}`;
  const encryptDestPath=`s3://dch-mes-mockb-dev-files-ec89/mestest/enc_${uniqueSysFilename}`;

  const queuesResponse = await sqs.listQueues();
  if (queuesResponse.urls.filter((q) => q === newFileQueue).length == 0) {
    fail(`New file queue was not found in the SQS queue listing`);
  }

  // Send file into MES IP
  await uploadFileToSftp("sftp.mockb.gamest.cloud", username, "", 22, fileData, uniqueSysFilename);
//  await uploadFileToTransfer(fileData, uniqueSysFilename);

  const timeFileUploaded = new Date().getTime();
  let newFileDestPath = await waitForNewFileMessage(newFileQueue, uniqueSysFilename);
  newFileTrend.add(new Date().getTime() - timeFileUploaded);


  // Decryption
  await sendDecryptionRequest(decryptRequestQueue, transactionIdToUseForVu, newFileDestPath, decryptDestPath, "GTRI", "UNKNOWN", decryptResponseQueue);

  const startDecryptTime = new Date().getTime();
  await waitForDecryptResponseMessage(decryptResponseQueue, transactionIdToUseForVu);
  decryptTrend.add(new Date().getTime() - startDecryptTime);
 
  // Encryption
  await sendEncryptionRequest(encryptRequestQueue, transactionIdToUseForVu, decryptDestPath, encryptDestPath, "GTRI", "UNKNOWN", encryptResponseQueue);

  const startEncryptTime = new Date().getTime();
  await waitForEncryptResponseMessage(encryptResponseQueue, transactionIdToUseForVu);
  encryptTrend.add(new Date().getTime() - startEncryptTime);

  // This sends to a REMOTE endpoint:
  const startWaitForDelivery = new Date().getTime();
  //  sendFileRequest(sendFileQueue, transactionIdToUseForVu, encryptDestPath, "testorgoutgoingdelivery");

  // This sends to a LOCAL org:
  await sendFileRequest(sendFileQueue, transactionIdToUseForVu, encryptDestPath, "testorglocal");


  // TODO: Now wait until the file exists in S3
  //      Maybe have it expect a specific file path, just check if that file exists directly?
  // s3://dch-mes-mockb-dev-sftp-mes-ip-test-user1/mes-ip-test-user1/loadtestoutgoing/enc_eld_gdc_daily_inbound_40321.txt.pgp
  await waitForS3FilePresence(localDeliveryBucketName, `${username}/loadtestoutgoing/`, uniqueSysFilename);
  s3deliveryTrend.add(new Date().getTime() - startWaitForDelivery);
  // END send_file
}

