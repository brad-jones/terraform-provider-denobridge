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
    // Create is given a set of props that should contain all required
    // information to create a new instance of the resource.

    // In this case we are just creating a file
    await Deno.writeTextFile(props.path, props.content);

    // At minimum, the create method must return an id that uniquely identifies this resource.
    // Optional additional state that is only known after the resource is created can be returned in the state object.
    return {
      id: props.path,
      state: {
        mtime: (await Deno.stat(props.path)).mtime!.getTime(),
      },
    };
  }

  async read(id: string, props: Props): Promise<{ props?: Props; state?: State; exists?: boolean }> {
    // Read is given the id of a previously created resource along with its props.
    // In this case the id is the same as the file path so we don't really need the
    // props to update the current state.

    // The read method's job is to get the current state of an existing resource.
    // And return an updated set of props & optional additional state that may
    // have changed (by outside actors), since the resource was initially created
    // (or last updated).
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
        // In the event we can no longer locate the resource we should signal
        // to Terraform to remove the resource from state. This will cause Terraform to re-create
        // the resource on the next plan.
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
    // Update is given the id of a previously created resource along with the
    // current props & state (this will either have come from the saved Terraform state
    // or more likely be the refreshed data returned from the read method),
    // as well as the next set of props to update the resource to.

    // In this case an in-place update is not supported for a file when the path
    // changes because that would mean the ID would change too & Terraform can't deal with that easily.
    // So we throw an error here but if an update wouldn't result in a new ID, there is no need to throw such an error.
    // Or implement the modifyPlan method, which should have caught this & told Terraform to do a replacement instead (ie: create then delete).
    if (nextProps.path !== currentProps.path) {
      throw new Error("Cannot change file path - requires resource replacement");
    }

    // Perform the update
    await Deno.writeTextFile(currentProps.path, nextProps.content);

    // Return the updated state
    return {
      mtime: (await Deno.stat(currentProps.path)).mtime!.getTime(),
    };
  }

  async delete(id: string, props: Props, state: State): Promise<void> {
    // Delete is given the id of a previously created resource along with its props & any additional computed state.

    // Perform the delete
    await Deno.remove(props.path);
  }

  // This method is optional. If not implemented, the provider will just carry on without making any plan modifications.
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
    // The modifyPlan method is given:
    //  - The id of the resource (which could be null if the resource is yet to be created)
    //  - nextProps will be set for create & update
    //  - currentProps will be set for update & delete
    //  - currentState will be set for update & delete

    // If you decide you don't want to make any changes to the plan, just return an empty object.
    if (!nextProps || !currentProps) {
      return {};
    }

    // The most common use case for this method is to tell Terraform if the resource
    // requires replacement (ie: create then delete) instead of an inline update.
    return {
      requiresReplacement: currentProps.path !== nextProps.path,
    };

    // Other use cases include returning a set of modifiedProps.
    // For example to provide default values for any unset props.
    // eg: return { modifiedProps: { ...nextProps, content: nextProps.content ?? "Hello World" } }

    // Or returning diagnostics, eg: for Destroy plans. As well as any other validation.
    // see: https://developer.hashicorp.com/terraform/plugin/framework/resources/plan-modification#resource-destroy-plan-diagnostics
    // eg: return {
    //   diagnostics: [{
    //     severity: "warning",
    //     summary: "Resource Destruction Considerations",
    //     detail: `Applying this resource destruction will only remove the resource
    //             from the Terraform state and will not call the deletion API due to API limitations.
    //             Manually use the web interface to fully destroy this resource.`
    //   }]
    // }
  }
}
