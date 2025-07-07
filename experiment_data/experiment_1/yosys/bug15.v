module top (in2, out7);

    input wire signed [1:0] in2;
    wire wire_3;
    wire signed wire_4;
    output wire out7;

    assign wire_3 = 1'b1;
    assign wire_4 = in2[0:0];
    assign out7 = (wire_3 ** (in2[1:1] ** wire_4));

endmodule