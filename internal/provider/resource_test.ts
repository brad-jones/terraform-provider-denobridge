import { DenoBridgeResource } from "@brad-jones/terraform-provider-denobridge";

interface Props {
  path: string;
  content: string;
}

interface State {
  mtime: number;
}

export default class FileResource extends DenoBridgeResource<Props, State, string> {
  async create(props: Props): Promise<{ id: string; state: State }> {
    await Deno.writeTextFile(props.path, props.content);

    return {
      id: props.path,
      state: {
        mtime: (await Deno.stat(props.path)).mtime!.getTime(),
      },
    };
  }

  async read(id: string, props: Props): Promise<{ props?: Props; state?: State; exists?: boolean }> {
    try {
      const content = await Deno.readTextFile(id);
      return {
        props: { path: id, content },
        state: {
          mtime: (await Deno.stat(id)).mtime!.getTime(),
        },
      };
    } catch (e) {
      if (e instanceof Deno.errors.NotFound) {
        return { exists: false };
      }
      throw e;
    }
  }

  async update(
    id: string,
    nextProps: Props,
    currentProps: Props,
    currentState: State,
  ): Promise<State> {
    if (nextProps.path !== currentProps.path) {
      throw new Error("Cannot change file path - requires resource replacement");
    }

    await Deno.writeTextFile(currentProps.path, nextProps.content);

    return {
      mtime: (await Deno.stat(currentProps.path)).mtime!.getTime(),
    };
  }

  async delete(id: string, props: Props, state: State): Promise<void> {
    await Deno.remove(props.path);
  }

  async modifyPlan(
    id: string | null,
    nextProps: Props | null,
    currentProps: Props | null,
    currentState: State | null,
  ): Promise<{
    modifiedProps?: Props;
    requiresReplacement?: boolean;
    diagnostics?: Array<{ severity: "error" | "warning"; summary: string; detail: string }>;
  }> {
    if (!nextProps || !currentProps) {
      return {};
    }

    return {
      requiresReplacement: currentProps.path !== nextProps.path,
    };
  }
}
