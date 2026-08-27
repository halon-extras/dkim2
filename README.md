# dkim2

### Experimental - currently using draft-ietf-dkim-dkim2-spec-02

Halon plugin for signing and verifying email with DKIM2 (using the [turscar/dkim2](https://pkg.go.dev/go.turscar.ie/dkim2) library). It provides HSL functions for generating `Message-Instance` and `DKIM2-Signature` headers and validating DKIM2-signed messages.

## Installation

Follow the [instructions](https://docs.halon.io/manual/comp_install.html#installation) in our manual to add our package repository and then run the below command.

### Ubuntu

```
apt-get install halon-extras-dkim2
```

### RHEL

```
yum install halon-extras-dkim2
```

### Azure Linux

```
tdnf install -y halon-extras-dkim2
```

## Exported functions

These functions needs to be [imported](https://docs.halon.io/hsl/structures.html#import) from the `extras://dkim2` module path.

### dkim2_sign(mail, mailfrom, rcptto, selector, domain, privatekey [, dkim2options])

Creates the `Message-Instance` and `DKIM2-Signature` headers for a message.

**Params**

- mail `File` - email file
- mailfrom `string` - envelope sender
- rcptto `array` - envelope recipients
- selector `string` - DNS key selector
- domain `string` - signing domain
- privatekey `PrivateKey|string` - PEM-encoded PKCS#8 Ed25519/RSA or PKCS#1 RSA private key
- dkim2options `array` - Optional signing options

The following options are available in the **dkim2options** array:

- `nonce` string - Optional signer-defined value placed in the `n=` tag. Limited to 64 printable ASCII characters and must not contain a semicolon
- `timestamp` number - Unix timestamp; the current time is used when omitted
- `exploded` boolean - Adds `exploded` to the `f=` tag, reporting that the message is being sent to more than one recipient. Defaults to `false`
- `donotexplode` boolean - Adds `donotexplode` to the `f=` tag, requesting that the message not be sent to more than one recipient. Defaults to `false`
- `donotmodify` boolean - Adds `donotmodify` to the `f=` tag, requesting that the message body and existing headers not be modified. Defaults to `false`
- `feedback` boolean - Adds `feedback` to the `f=` tag, requesting feedback about how the message is handled during and after delivery. Defaults to `false`

These options are encoded into and protected by the generated `DKIM2-Signature` header. See [sections 7.3 and 7.9 of the DKIM2 specification](https://datatracker.ietf.org/doc/html/draft-ietf-dkim-dkim2-spec-02#section-7.3).

**Returns**

An associative array with `result`, `error`, and `headers` properties. On success, `headers` contains complete header fields, including their trailing CRLF, in the order in which they must be prepended to the original message.

**Example**

```
import { dkim2_sign } from "extras://dkim2";

$mail = MailMessage::File(File("simple_email.eml"));
$privatekey = File::read("dkim2_ed25519_test_key.private");
$rcptto = ["recipient@example.com"];

$dkim2options = [
    "timestamp" => time(),
];

$signed = dkim2_sign(
    $mail->toFile(),
    "sender@test1.dkim2.com",
    $rcptto,
    "ed25519",
    "test1.dkim2.com",
    $privatekey,
    $dkim2options
);

if (!$signed["result"]) 
{
    throw Exception($signed["error"]);
}

foreach ($signed["headers"] as $header)
    $mail->modifyContent(0, 0, $header);
echo $mail->toString();
```

### dkim2_verify(mail, mailfrom, rcptto[, dkim2options])

Verify a `DKIM2-Signature`.

**Params**

- mail `File` - email file
- mailfrom `string` - Sender of the email
- rcptto `array` - Array of recievers
- dkim2options `array` - Optional DKIM2 options

The following options are available in the **dkim2options** array

- `ignoretimestamp` - Reject anything with a timestamp more than 14 days old. Defaults to `false`
- `timeout` - Maximum time in seconds allowed for DNS key lookups during verification. Defaults to `5`

**Returns**

An associative array with a `result` and `error` property. `error` is set if an error occurs.

**Example**

```
import { dkim2_verify } from "extras://dkim2";

$mail = MailMessage::File(File("simple_email.eml"));

$rcptto = ["recipient@example.com"];

$dkim2options = [
    "ignoretimestamp" => true,
];

$verified = dkim2_verify($mail->toFile(), "sender@test1.dkim2.com", $rcptto, $dkim2options);

if (!$verified["result"]) 
{
    echo "dkim2 signature not found";
}

else if ($verified["result"] != "pass")
{
    throw $verified["error"];
} 
else 
{
    echo "verfied ok!";
}

```
