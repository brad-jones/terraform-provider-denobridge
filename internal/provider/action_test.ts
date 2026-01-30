import { DenoBridgeAction, type JSONRPCClient } from "@brad-jones/terraform-provider-denobridge";

interface Props {
  path: string;
  content: string;
}

export default class WriteFileAction extends DenoBridgeAction<Props, void> {
  async invoke(props: Props, client: JSONRPCClient): Promise<void> {
    await client.invokeProgress({ message: "about to write file" });
    await Deno.writeTextFile(props.path, props.content);
    await client.invokeProgress({ message: "file written" });
  }
}
