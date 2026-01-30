import { DenoBridgeAction, type JSONRPCClient } from "@brad-jones/terraform-provider-denobridge";

interface Props {
  destination: string;
}

interface Result {
  launched: boolean;
  timestamp: string;
}

export default class RocketLaunchAction extends DenoBridgeAction<Props, Result> {
  async invoke(props: Props, client: JSONRPCClient): Promise<Result> {
    // Report progress to the user during the action
    await client.invokeProgress({ message: `Launching rocket to ${props.destination}` });
    await new Promise((resolve) => setTimeout(resolve, 500));

    await client.invokeProgress({ message: "3..." });
    await new Promise((resolve) => setTimeout(resolve, 500));

    await client.invokeProgress({ message: "2..." });
    await new Promise((resolve) => setTimeout(resolve, 500));

    await client.invokeProgress({ message: "1..." });
    await new Promise((resolve) => setTimeout(resolve, 500));

    await client.invokeProgress({ message: "Blast off!" });

    return {
      launched: true,
      timestamp: new Date().toISOString(),
    };
  }
}
