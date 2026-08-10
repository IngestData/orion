namespace IngestData.Orion.Input;

using IngestData.Ingress.Transformer;
using System;
using System.Collections.Generic;

public class Transformer : TransformerBase
{
    public string Name { get; set; }
    public string Type { get; set; }
    public Dictionary<string, object> Parameters { get; set; }

    public dynamic Input { get; set; }

    public Transformer(string name, string type, Dictionary<string, object> parameters, dynamic input)
    {
        Name = name;
        Type = type;
        Parameters = parameters;
        Input = input;
    }

  public override dynamic Transform(dynamic input)
    {
        // Implement the transformation logic based on the Type and Parameters
        // For example, if Type is "uppercase", convert the input to uppercase
        if (Type == "uppercase")
        {
            return input.ToString().ToUpper();
        }
        else if (Type == "lowercase")
        {
            return input.ToString().ToLower();
        }
        else if (Type == "replace" && Parameters.ContainsKey("old") && Parameters.ContainsKey("new"))
        {
            string oldValue = Parameters["old"].ToString();
            string newValue = Parameters["new"].ToString();
            return input.ToString().Replace(oldValue, newValue);
        }
        else
        {
            throw new NotImplementedException($"Transformation type '{Type}' is not implemented.");
        }
    }
}
