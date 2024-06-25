const axios = require('axios');
const https = require('https');

const BASE_URL = '<API Gateway URL here>';
const USERNAME = '<App Cred Username>';
const PASSWORD = '<App Cred Password>';

const INSTANCE_ADDRESS = '<Deployed Smart contract address>';
const FROM_ADDRESS = '<Wallet Address>';

const MAX_SOCKETS = 100;
const MAX_PROMISES = 100;
const TOTAL_MESSAGES = 10000;

const client = axios.create({
  httpsAgent: new https.Agent({
    maxSockets: MAX_SOCKETS,
  }),
  auth: {
    username: USERNAME,
    password: PASSWORD,
  },
  baseURL: BASE_URL
});

async function run() {

  let i = 0;
  const inflight = {};
  while (i < TOTAL_MESSAGES) {
    if (Object.keys(inflight).length < MAX_PROMISES) {
      const reqNo = ++i;
      inflight[`${reqNo}`] = (client.request({
        url: `instances/${INSTANCE_ADDRESS}/set`,
        method: 'POST',
        data: {
          "x": reqNo%100,
        },
        // data: {
        //   "someMessage": `Test ${reqNo}`,
        //   "someNumber": reqNo,
        // },
        params: {
          'kld-from': FROM_ADDRESS,
          'kld-sync': 'false',
          // 'kld-acktype': 'receipt',
        }
      })
        .then(({status})  => console.log(`PASS - ${reqNo} [${status}]`))
        .catch(err => console.log(`FAIL - ${reqNo} [${err.response && err.response.status}]: ${(err.response && err.response.data && JSON.stringify(err.response.data)) || err.message}`))
        .then(() => reqNo)
      );
    } else {
      const reqNo = await Promise.any(Object.values(inflight))
      delete inflight[reqNo]
    }
  }

}

run().catch(err => {
  console.error(err.stack);
  process.exit(1);
});