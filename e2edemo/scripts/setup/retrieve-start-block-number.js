const url = require('url');
const rlp = require('rlp');
(() => {
  if (process.argv.length < 5) {
    return console.error('invalid arguments, node ./retrieve-start-block-number.js {ICON-NODE-ENDPOINT} {BTP ADDRESS OF BMC ON BSC} {ICON CONTRACT ADDRESS}');
  }

  const [ endpoint, src, dst ] = process.argv.slice(2);
  const protocol = url.parse(endpoint).protocol.replace(':', '');
  let client;
  switch (protocol) {
    case 'https': {
      client = require('https');
      break;
    }
    case 'http': {
      client = require('http');
      break;
    }
    default: {
      throw new Error('not supports protocol:', protocol);
    }
  }

  const req = client.request(endpoint, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    }
  }, (res) => {
    res.on('data', (chunk) => {
      console.log('start-number:', Number.parseInt(rlp.decode(Buffer.from(JSON.parse(chunk).result.verifier.extra.slice(2), 'hex'))[0].toString('hex'), 16));
    });
  });

  req.on('error', console.error);

  req.write(JSON.stringify({
    jsonrpc: '2.0',
    method: 'icx_call',
    id: 1,
    params: {
      to: dst,
      dataType: 'call',
      data: {
        method: 'getStatus',
        params: {
          _link: src
        }
      }
    }
  }));
  req.end();
})();
