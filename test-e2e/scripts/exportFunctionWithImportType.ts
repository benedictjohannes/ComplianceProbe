import type { ScriptContext } from "crobe-sdk/func";

export default ({ os }: ScriptContext) => `echo 'hello from ${os}'`;
